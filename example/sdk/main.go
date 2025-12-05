package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/beanbocchi/templar/pkg/sdk"
	"github.com/google/uuid"
)

func main() {
	client := sdk.NewClient("http://localhost:8080/api/v1")

	// Push
	// Prepare file content
	fileContent := bytes.NewBufferString("This is template content")

	// Upload template
	templateID := uuid.New()
	resp, err := client.Push(sdk.PushRequest{
		TemplateID: templateID,
		Version:    1,
		File:       fileContent,
		FileName:   "template.txt",
	})
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}

	fmt.Printf("Upload successful: %s\n", resp.Message)

	// Pull
	fileReader, err := os.Create("downloaded_template.txt")
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer fileReader.Close()

	err = client.Pull(sdk.PullRequest{
		TemplateID: templateID,
		Version:    1,
	}, fileReader)
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		return
	}

	fmt.Println("Download successful")
}
