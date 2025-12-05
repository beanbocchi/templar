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

	client := &ClientImpl{
		project:     project,
		bucket:      cfg.Bucket,
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		accessGrant: cfg.AccessGrant,
	}

	return client, nil
}

// Close closes the Storj project connection
func (c *ClientImpl) Close() error {
	if c.project != nil {
		return c.project.Close()
	}
	return nil
}

// CreateMultipart starts a Storj multipart upload using BeginUpload and
// returns the uploadID Storj generates.
func (c *ClientImpl) CreateMultipart(ctx context.Context, key string) (string, error) {
	info, err := c.project.BeginUpload(ctx, c.bucket, key, nil)
	if err != nil {
		return "", fmt.Errorf("begin upload: %w", err)
	}

	return info.UploadID, nil
}

// UploadPart uploads a single part to Storj using the native multipart API.
func (c *ClientImpl) UploadPart(
	ctx context.Context,
	key string,
	uploadID string,
	partNumber int,
	content io.Reader,
) error {
	pu, err := c.project.UploadPart(ctx, c.bucket, key, uploadID, uint32(partNumber))
	if err != nil {
		return fmt.Errorf("begin part upload: %w", err)
	}

	// Stream data into the part.
	if _, err := io.Copy(pu, content); err != nil {
		_ = pu.Abort()
		return fmt.Errorf("write part: %w", err)
	}

	// Commit the part. Storj will record size/metadata internally.
	if err := pu.Commit(); err != nil {
		return fmt.Errorf("commit part: %w", err)
	}

	return nil
}

// CompleteMultipart finalizes the multipart upload in Storj.
func (c *ClientImpl) CompleteMultipart(
	ctx context.Context,
	key string,
	uploadID string,
) error {
	if _, err := c.project.CommitUpload(ctx, c.bucket, key, uploadID, nil); err != nil {
		return fmt.Errorf("commit upload: %w", err)
	}

	return nil
}

// AbortMultipart cancels the multipart upload in Storj.
func (c *ClientImpl) AbortMultipart(ctx context.Context, key, uploadID string) error {
	if err := c.project.AbortUpload(ctx, c.bucket, key, uploadID); err != nil {
		return fmt.Errorf("abort upload: %w", err)
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
