# Templar SDK

Templar SDK is a Go client library for interacting with the Templar API.

## Installation

```bash
go get github.com/beanbocchi/templar/pkg/sdk
```

## Quick Start

```go
package main

import (
    "fmt"
    "io"
    "os"
    
    "github.com/beanbocchi/templar/pkg/sdk"
    "github.com/google/uuid"
)

func main() {
    // Create client
    client := sdk.NewClient("http://localhost:8080/api/v1")
    
    // Push - Upload template
    templateID := uuid.New()
    file, _ := os.Open("template.txt")
    defer file.Close()
    
    pushResp, err := client.Push(sdk.PushRequest{
        TemplateID: templateID,
        Version:    1,
        File:       file,
        FileName:   "template.txt",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println("Upload successful:", pushResp.Message)
    
    // Pull - Download template
    fileReader, err := client.Pull(sdk.PullRequest{
        TemplateID: templateID,
        Version:    1,
    })
    if err != nil {
        panic(err)
    }
    defer fileReader.Close()
    
    // Save file
    output, _ := os.Create("downloaded_template.txt")
    defer output.Close()
    io.Copy(output, fileReader)
    
    // ListTemplates - List all templates
    templates, err := client.ListTemplates(&sdk.ListTemplatesRequest{
        Search: "keyword", // Optional
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Found %d templates\n", len(templates))
    
    // ListVersions - List all versions of a template
    versions, err := client.ListVersions(sdk.ListVersionsRequest{
        TemplateID: templateID,
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Found %d versions\n", len(versions))
    
    // ListJobs - List all jobs
    jobs, err := client.ListJobs(&sdk.ListJobsRequest{
        Page:  int32Ptr(1),
        Limit: int32Ptr(10),
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Found %d jobs\n", len(jobs))
}

func int32Ptr(i int32) *int32 {
    return &i
}
```

## API Documentation

### Client

#### NewClient(baseURL string) *Client

Creates a new SDK client.

- `baseURL`: The base URL of the API, e.g., "http://localhost:8080/api/v1"

#### NewClientWithHTTPClient(baseURL string, httpClient *http.Client) *Client

Creates an SDK client with a custom HTTP client.

### Push

Upload a template file to the server.

```go
func (c *Client) Push(req PushRequest) (*PushResponse, error)
```

**Parameters:**
- `TemplateID`: Template ID (uuid.UUID)
- `Version`: Version number (int64, must be >= 1)
- `File`: File content (io.Reader)
- `FileName`: File name (string, optional)

**Returns:**
- `*PushResponse`: Response containing success message
- `error`: Error information

### Pull

Download a template file from the server.

```go
func (c *Client) Pull(req PullRequest) (io.ReadCloser, error)
```

**Parameters:**
- `TemplateID`: Template ID (uuid.UUID)
- `Version`: Version number (int64, must be >= 1)

**Returns:**
- `io.ReadCloser`: Reader for file content, caller is responsible for closing
- `error`: Error information

### ListTemplates

List all templates.

```go
func (c *Client) ListTemplates(req *ListTemplatesRequest) ([]Template, error)
```

**Parameters:**
- `Search`: Optional search keyword (string)

**Returns:**
- `[]Template`: List of templates
- `error`: Error information

### ListVersions

List all versions of a specified template.

```go
func (c *Client) ListVersions(req ListVersionsRequest) ([]TemplateVersion, error)
```

**Parameters:**
- `TemplateID`: Template ID (uuid.UUID)

**Returns:**
- `[]TemplateVersion`: List of versions
- `error`: Error information

### ListJobs

List all jobs.

```go
func (c *Client) ListJobs(req *ListJobsRequest) ([]Job, error)
```

**Parameters:**
- `Page`: Page number, starts from 1 (*int32, optional)
- `Limit`: Number of items per page, max 100 (*int32, optional)
- `Cursor`: Cursor for cursor-based pagination (*string, optional)

**Returns:**
- `[]Job`: List of jobs
- `error`: Error information

## Error Handling

The SDK uses the `sdk.Error` type to represent API errors. This type implements the `error` interface and contains an error code and error message.

```go
if err != nil {
    if sdkErr, ok := err.(*sdk.Error); ok {
        fmt.Printf("Error code: %s, Error message: %s\n", sdkErr.ErrCode, sdkErr.Message)
    } else {
        fmt.Printf("Other error: %v\n", err)
    }
}
```

## Examples

For more examples, please refer to the `example/` directory.
