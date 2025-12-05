package sdk

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/beanbocchi/templar/pkg/response"
	"github.com/google/uuid"
	"github.com/zeebo/blake3"
)

// Client is the Templar SDK client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new SDK client
// baseURL is the base URL of the API, e.g., "http://localhost:8080/api/v1"
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithHTTPClient creates an SDK client with a custom HTTP client
func NewClientWithHTTPClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// PushRequest is the request parameters for Push
type PushRequest struct {
	TemplateID uuid.UUID
	Version    int64
	File       io.Reader
	FileName   string
}

// PushResponse is the response from Push
type PushResponse struct {
	Message string `json:"message"`
	Hash    string `json:"hash,omitempty"`
}

// Push uploads a template file to the server
func (c *Client) Push(req PushRequest) (*PushResponse, error) {
	// Stream multipart to avoid buffering whole file in memory.
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	hasher := blake3.New()
	writeErr := make(chan error, 1)

	fileName := req.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("template_%s_%d", req.TemplateID.String(), req.Version)
	}

	go func() {
		defer close(writeErr)
		defer pw.Close()

		if err := writer.WriteField("template_id", req.TemplateID.String()); err != nil {
			pw.CloseWithError(err)
			writeErr <- fmt.Errorf("write template_id: %w", err)
			return
		}

		if err := writer.WriteField("version", fmt.Sprintf("%d", req.Version)); err != nil {
			pw.CloseWithError(err)
			writeErr <- fmt.Errorf("write version: %w", err)
			return
		}

		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			pw.CloseWithError(err)
			writeErr <- fmt.Errorf("create form file: %w", err)
			return
		}

		// Hash while streaming to minimize RAM usage.
		if _, err := io.Copy(part, io.TeeReader(req.File, hasher)); err != nil {
			pw.CloseWithError(err)
			writeErr <- fmt.Errorf("copy file: %w", err)
			return
		}

		if err := writer.Close(); err != nil {
			pw.CloseWithError(err)
			writeErr <- fmt.Errorf("close writer: %w", err)
			return
		}
	}()

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/push", c.baseURL), pr)
	if err != nil {
		<-writeErr
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Ensure writer goroutine finishes.
		<-writeErr
		return nil, err
	}
	defer resp.Body.Close()

	if wErr := <-writeErr; wErr != nil {
		return nil, wErr
	}

	// Decode response
	var commonResp response.CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&commonResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest && commonResp.Error == nil {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}
	if commonResp.Error != nil {
		return nil, commonResp.Error
	}

	var pushResp PushResponse
	dataBytes, err := json.Marshal(commonResp.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}
	if err := json.Unmarshal(dataBytes, &pushResp); err != nil {
		if msg, ok := commonResp.Data.(string); ok {
			pushResp.Message = msg
		} else {
			return nil, fmt.Errorf("unmarshal PushResponse: %w", err)
		}
	}

	// Include computed hash (client-side) for caller convenience.
	pushResp.Hash = hex.EncodeToString(hasher.Sum(nil))

	return &pushResp, nil
}

func (c *Client) getHash(templateID uuid.UUID, version int64) (string, error) {
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s/versions/%s/%d", c.baseURL, templateID.String(), version), nil)
	if err != nil {
		return "", fmt.Errorf("create get hash request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send get hash request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var commonResp response.CommonResponse
		if err := json.NewDecoder(resp.Body).Decode(&commonResp); err == nil && commonResp.Error != nil {
			return "", commonResp.Error
		}

		data := commonResp.Data.(map[string]any)
		if fileHash, ok := data["file_hash"]; ok {
			return fileHash.(string), nil
		}
		return "", fmt.Errorf("file hash not found")
	}

	return "", fmt.Errorf("get hash failed with status %d", resp.StatusCode)
}

// PullRequest is the request parameters for Pull
type PullRequest struct {
	TemplateID uuid.UUID `json:"template_id"`
	Version    int64     `json:"version"`
}

// Pull streams a template to dst with minimal buffering
func (c *Client) Pull(req PullRequest, dst io.Writer) error {
	expectedHash, err := c.getHash(req.TemplateID, req.Version)
	if err != nil {
		return fmt.Errorf("get hash: %w", err)
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal pull request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/pull", c.baseURL), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create pull request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send pull request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var commonResp response.CommonResponse
		if err := json.NewDecoder(resp.Body).Decode(&commonResp); err == nil && commonResp.Error != nil {
			return commonResp.Error
		}
		return fmt.Errorf("pull failed with status %d", resp.StatusCode)
	}

	reader := resp.Body

	var writer io.Writer = dst
	var hasher *blake3.Hasher
	if expectedHash != "" {
		h := blake3.New()
		hasher = h
		writer = io.MultiWriter(dst, h)
	}

	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("stream download: %w", err)
	}

	if hasher != nil {
		got := hex.EncodeToString(hasher.Sum(nil))
		if got != expectedHash {
			return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, got)
		}
	}

	return nil
}
