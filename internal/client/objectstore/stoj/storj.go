package stoj

import (
	"context"
	"fmt"
	"io"
	"strings"

	"storj.io/uplink"
)

type ClientImpl struct {
	project     *uplink.Project
	bucket      string
	baseURL     string
	accessGrant string
}

type StorjConfig struct {
	// AccessGrant is the Storj access grant string
	AccessGrant string
	// Bucket is the bucket name where objects will be stored
	Bucket string
	// BaseURL is the public base URL used to construct public URLs
	// If empty, GetURL will return an error for public objects
	BaseURL string
}

// NewClient creates a new Storj objectstore client
func NewClient(ctx context.Context, cfg StorjConfig) (*ClientImpl, error) {
	if cfg.AccessGrant == "" {
		return nil, fmt.Errorf("access grant is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Parse the access grant
	access, err := uplink.ParseAccess(cfg.AccessGrant)
	if err != nil {
		return nil, fmt.Errorf("parse access grant: %w", err)
	}

	// Open project directly
	project, err := uplink.OpenProject(ctx, access)
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}

	// Ensure bucket exists
	_, err = project.EnsureBucket(ctx, cfg.Bucket)
	if err != nil {
		project.Close()
		return nil, fmt.Errorf("ensure bucket: %w", err)
	}

	return &ClientImpl{
		project:     project,
		bucket:      cfg.Bucket,
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		accessGrant: cfg.AccessGrant,
	}, nil
}

// Close closes the Storj project connection
func (c *ClientImpl) Close() error {
	if c.project != nil {
		return c.project.Close()
	}
	return nil
}

// Upload uploads an object to Storj
func (c *ClientImpl) Upload(ctx context.Context, key string, content io.Reader) error {
	// Start upload
	upload, err := c.project.UploadObject(ctx, c.bucket, key, nil)
	if err != nil {
		return fmt.Errorf("initiate upload: %w", err)
	}

	// Copy data to upload
	_, err = io.Copy(upload, content)
	if err != nil {
		upload.Abort()
		return fmt.Errorf("write data: %w", err)
	}

	// Commit the upload
	err = upload.Commit()
	if err != nil {
		return fmt.Errorf("commit upload: %w", err)
	}

	return nil
}

// Download downloads an object from Storj
func (c *ClientImpl) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	download, err := c.project.DownloadObject(ctx, c.bucket, key, nil)
	if err != nil {
		return nil, fmt.Errorf("download object: %w", err)
	}

	return download, nil
}

// Delete deletes an object from Storj
func (c *ClientImpl) Delete(ctx context.Context, key string) error {
	_, err := c.project.DeleteObject(ctx, c.bucket, key)
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}
