package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/beanbocchi/templar/pkg/response"
)

func (c *Client) doGET(path string, query map[string]string) (*response.CommonResponse, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if len(query) > 0 {
		q := httpReq.URL.Query()
		for key, value := range query {
			if value == "" {
				continue
			}
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	return c.doRequest(httpReq)
}

func (c *Client) doPOST(path string, body io.Reader, contentType string) (*response.CommonResponse, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	httpReq, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	return c.doRequest(httpReq)
}

func (c *Client) doRequest(req *http.Request) (*response.CommonResponse, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var commonResp response.CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&commonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest && commonResp.Error == nil {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	if commonResp.Error != nil {
		return nil, commonResp.Error
	}

	return &commonResp, nil
}
