package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
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

// CommonResponse is the common API response format
type CommonResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error *Error      `json:"error,omitempty"`
}

// Error represents an API error
type Error struct {
	ErrCode string `json:"err_code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.ErrCode, e.Message)
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
}

// Push uploads a template file to the server
func (c *Client) Push(req PushRequest) (*PushResponse, error) {
	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add template_id
	if err := writer.WriteField("template_id", req.TemplateID.String()); err != nil {
		return nil, fmt.Errorf("failed to write template_id: %w", err)
	}

	// Add version
	if err := writer.WriteField("version", fmt.Sprintf("%d", req.Version)); err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}

	// Add file
	fileName := req.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("template_%s_%d", req.TemplateID.String(), req.Version)
	}
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file field: %w", err)
	}

	if _, err := io.Copy(part, req.File); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Send request
	url := fmt.Sprintf("%s/push", c.baseURL)
	httpReq, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var commonResp CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&commonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if commonResp.Error != nil {
		return nil, commonResp.Error
	}

	// Parse data field
	var pushResp PushResponse
	dataBytes, err := json.Marshal(commonResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &pushResp); err != nil {
		// If data is a string, use it directly
		if msg, ok := commonResp.Data.(string); ok {
			pushResp.Message = msg
		} else {
			return nil, fmt.Errorf("failed to unmarshal PushResponse: %w", err)
		}
	}

	return &pushResp, nil
}

// PullRequest is the request parameters for Pull
type PullRequest struct {
	TemplateID uuid.UUID
	Version    int64
}

// Pull downloads a template file from the server
// Returns a Reader for the file content, the caller is responsible for closing it
func (c *Client) Pull(req PullRequest) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/pull", c.baseURL)

	// Create request body
	requestBody := map[string]interface{}{
		"template_id": req.TemplateID.String(),
		"version":     req.Version,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		// Try to parse error response
		var commonResp CommonResponse
		if err := json.NewDecoder(resp.Body).Decode(&commonResp); err == nil && commonResp.Error != nil {
			return nil, commonResp.Error
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// Template represents a template
type Template struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// ListTemplatesRequest is the request parameters for ListTemplates
type ListTemplatesRequest struct {
	Search string // Optional search keyword
}

// ListTemplates lists all templates
func (c *Client) ListTemplates(req *ListTemplatesRequest) ([]Template, error) {
	url := fmt.Sprintf("%s/templates", c.baseURL)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	if req != nil && req.Search != "" {
		q := httpReq.URL.Query()
		q.Add("search", req.Search)
		httpReq.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var commonResp CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&commonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if commonResp.Error != nil {
		return nil, commonResp.Error
	}

	// Parse data field
	var templates []Template
	dataBytes, err := json.Marshal(commonResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Template list: %w", err)
	}

	return templates, nil
}

// TemplateVersion represents a template version
type TemplateVersion struct {
	ID            string  `json:"id"`
	TemplateID    string  `json:"template_id"`
	VersionNumber int64   `json:"version_number"`
	ObjectKey     string  `json:"object_key"`
	FileSize      *int64  `json:"file_size"`
	FileHash      *string `json:"file_hash"`
	CreatedAt     string  `json:"created_at"`
}

// ListVersionsRequest is the request parameters for ListVersions
type ListVersionsRequest struct {
	TemplateID uuid.UUID
}

// ListVersions lists all versions of a specified template
func (c *Client) ListVersions(req ListVersionsRequest) ([]TemplateVersion, error) {
	url := fmt.Sprintf("%s/versions", c.baseURL)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := httpReq.URL.Query()
	q.Add("template_id", req.TemplateID.String())
	httpReq.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var commonResp CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&commonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if commonResp.Error != nil {
		return nil, commonResp.Error
	}

	// Parse data field
	var versions []TemplateVersion
	dataBytes, err := json.Marshal(commonResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &versions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TemplateVersion list: %w", err)
	}

	return versions, nil
}

// Job represents a job
type Job struct {
	ID            int64   `json:"id"`
	Type          string  `json:"type"`
	TemplateID    string  `json:"template_id"`
	VersionNumber *int64  `json:"version_number"`
	Status        string  `json:"status"`
	Progress      int64   `json:"progress"`
	StartedAt     string  `json:"started_at"`
	CompletedAt   *string `json:"completed_at"`
	ErrorMessage  *string `json:"error_message"`
	Metadata      string  `json:"metadata"`
}

// ListJobsRequest is the request parameters for ListJobs
type ListJobsRequest struct {
	Page   *int32  // Page number (starts from 1)
	Limit  *int32  // Number of items per page (max 100)
	Cursor *string // Cursor for cursor-based pagination
}

// ListJobs lists all jobs
func (c *Client) ListJobs(req *ListJobsRequest) ([]Job, error) {
	url := fmt.Sprintf("%s/jobs", c.baseURL)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	if req != nil {
		q := httpReq.URL.Query()
		if req.Page != nil {
			q.Add("page", fmt.Sprintf("%d", *req.Page))
		}
		if req.Limit != nil {
			q.Add("limit", fmt.Sprintf("%d", *req.Limit))
		}
		if req.Cursor != nil {
			q.Add("cursor", *req.Cursor)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var commonResp CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&commonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if commonResp.Error != nil {
		return nil, commonResp.Error
	}

	// Parse data field
	var jobs []Job
	dataBytes, err := json.Marshal(commonResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &jobs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Job list: %w", err)
	}

	return jobs, nil
}
