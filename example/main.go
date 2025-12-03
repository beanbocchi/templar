package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const baseURL = "http://localhost:8080/api/v1"

type PullRequest struct {
	TemplateID uuid.UUID `json:"template_id"`
	Version    int64     `json:"version"`
}

func main() {
	templateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	fmt.Println("=== Push Example ===")
	if err := pushTemplate(templateID, 1, "example/template.txt"); err != nil {
		fmt.Printf("Push error: %v\n", err)
	} else {
		fmt.Println("Push successful!")
	}

	time.Sleep(1 * time.Second)

	fmt.Println("\n=== Pull Example ===")
	if err := pullTemplate(templateID, 1, "downloaded_template.txt"); err != nil {
		fmt.Printf("Pull error: %v\n", err)
	} else {
		fmt.Println("Pull successful!")
	}
}

func pushTemplate(templateID uuid.UUID, version int64, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("template_id", templateID.String()); err != nil {
		return fmt.Errorf("failed to write template_id field: %w", err)
	}

	if err := writer.WriteField("version", fmt.Sprintf("%d", version)); err != nil {
		return fmt.Errorf("failed to write version field: %w", err)
	}

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/push", &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nResponse: %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func pullTemplate(templateID uuid.UUID, version int64, outputPath string) error {
	reqBody := PullRequest{
		TemplateID: templateID,
		Version:    version,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/pull", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Downloaded %d bytes to %s\n", written, outputPath)
	return nil
}
