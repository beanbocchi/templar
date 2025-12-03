package sdk_test

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/beanbocchi/templar/pkg/sdk"
	"github.com/google/uuid"
)

func ExampleClient_Push() {
	// Create client
	client := sdk.NewClient("http://localhost:8080/api/v1")

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
}

func ExampleClient_Pull() {
	// Create client
	client := sdk.NewClient("http://localhost:8080/api/v1")

	// Download template
	templateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	fileReader, err := client.Pull(sdk.PullRequest{
		TemplateID: templateID,
		Version:    1,
	})
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		return
	}
	defer fileReader.Close()

	// Save to file
	output, err := os.Create("downloaded_template.txt")
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer output.Close()

	if _, err := io.Copy(output, fileReader); err != nil {
		fmt.Printf("Failed to save file: %v\n", err)
		return
	}

	fmt.Println("Download successful")
}

func ExampleClient_ListTemplates() {
	// Create client
	client := sdk.NewClient("http://localhost:8080/api/v1")

	// List all templates (with search)
	templates, err := client.ListTemplates(&sdk.ListTemplatesRequest{
		Search: "keyword",
	})
	if err != nil {
		fmt.Printf("Failed to list templates: %v\n", err)
		return
	}

	fmt.Printf("Found %d templates\n", len(templates))
	for _, template := range templates {
		fmt.Printf("- %s: %s\n", template.ID, template.Name)
	}
}

func ExampleClient_ListVersions() {
	// Create client
	client := sdk.NewClient("http://localhost:8080/api/v1")

	// List all versions of a template
	templateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	versions, err := client.ListVersions(sdk.ListVersionsRequest{
		TemplateID: templateID,
	})
	if err != nil {
		fmt.Printf("Failed to list versions: %v\n", err)
		return
	}

	fmt.Printf("Found %d versions\n", len(versions))
	for _, version := range versions {
		fmt.Printf("- Version %d: %s\n", version.VersionNumber, version.ID)
	}
}

func ExampleClient_ListJobs() {
	// Create client
	client := sdk.NewClient("http://localhost:8080/api/v1")

	// List jobs (with pagination)
	page := int32(1)
	limit := int32(10)
	jobs, err := client.ListJobs(&sdk.ListJobsRequest{
		Page:  &page,
		Limit: &limit,
	})
	if err != nil {
		fmt.Printf("Failed to list jobs: %v\n", err)
		return
	}

	fmt.Printf("Found %d jobs\n", len(jobs))
	for _, job := range jobs {
		fmt.Printf("- Job %d: %s (Status: %s, Progress: %d%%)\n",
			job.ID, job.Type, job.Status, job.Progress)
	}
}
