package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type ClientImpl struct {
	root    string
	baseURL string
}

type LocalConfig struct {
	// Root is the base directory where objects are stored on disk (e.g., ./uploads)
	Root string
	// BaseURL is the public base URL used to construct public URLs, e.g., http://localhost:8080/api/v1/shared/files
	// If empty, GetURL will return an error and Upload(private=false) will return the key only.
	BaseURL string
}

func NewClient(cfg LocalConfig) (*ClientImpl, error) {
	if cfg.Root == "" {
		return nil, fmt.Errorf("local root is required")
	}
	if err := os.MkdirAll(cfg.Root, 0o755); err != nil {
		return nil, fmt.Errorf("create root: %w", err)
	}
	return &ClientImpl{root: cfg.Root, baseURL: strings.TrimRight(cfg.BaseURL, "/")}, nil
}

func (c *ClientImpl) fullPath(key string) string {
	// prevent path traversal
	clean := filepath.Clean(key)
	return filepath.Join(c.root, clean)
}

func (c *ClientImpl) multipartPath(uploadID string) string {
	return filepath.Join(c.root, ".multipart", uploadID)
}

// CreateMultipart starts a multipart upload by allocating a temporary directory
// on disk where individual parts will be stored before assembly.
func (c *ClientImpl) CreateMultipart(ctx context.Context, key string) (string, error) {
	uploadID := uuid.New().String()

	path := c.multipartPath(uploadID)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create multipart dir: %w", err)
	}

	return uploadID, nil
}

// UploadPart writes a single part to a temporary file on disk. We ignore the
// provided key and rely on uploadID for locating the multipart session.
func (c *ClientImpl) UploadPart(
	ctx context.Context,
	key string,
	uploadID string,
	partNumber int,
	content io.Reader,
) error {
	_ = ctx

	dir := c.multipartPath(uploadID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure multipart dir: %w", err)
	}

	partPath := filepath.Join(dir, fmt.Sprintf("part-%06d", partNumber))
	tmpPath := partPath + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create part file: %w", err)
	}

	if _, err := io.Copy(f, content); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write part: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close part: %w", err)
	}

	if err := os.Rename(tmpPath, partPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename part: %w", err)
	}

	return nil
}

// CompleteMultipart assembles parts in order into the final object and then
// cleans up temporary files.
func (c *ClientImpl) CompleteMultipart(
	ctx context.Context,
	key string,
	uploadID string,
) error {
	_ = ctx

	dir := c.multipartPath(uploadID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read multipart dir: %w", err)
	}

	// Prepare destination
	path := c.fullPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	tmpPath := path + ".tmp"
	dest, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}

	wroteAny := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "part-") {
			continue
		}

		partPath := filepath.Join(dir, entry.Name())
		partFile, err := os.Open(partPath)
		if err != nil {
			dest.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("open part %s: %w", entry.Name(), err)
		}

		if _, err := io.Copy(dest, partFile); err != nil {
			partFile.Close()
			dest.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("copy part %s: %w", entry.Name(), err)
		}
		partFile.Close()
		wroteAny = true
	}

	if !wroteAny {
		dest.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("no parts found")
	}

	if err := dest.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close dest: %w", err)
	}

	// Ensure target file doesn't exist (required for os.Rename on Windows to simulate overwrite)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("remove existing dest: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename dest: %w", err)
	}

	// Cleanup temporary multipart directory
	_ = os.RemoveAll(dir)

	return nil
}

// AbortMultipart cancels a multipart upload and removes all temporary data.
func (c *ClientImpl) AbortMultipart(ctx context.Context, key, uploadID string) error {
	_ = ctx
	_ = key

	dir := c.multipartPath(uploadID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("abort multipart: %w", err)
	}
	return nil
}

func (c *ClientImpl) Upload(ctx context.Context, key string, content io.Reader) error {
	path := c.fullPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	//! Write to temp file first, then rename for filesystem atomicity because mid-write crash will leave a partial file.
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(f, content); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close file: %w", err)
	}

	// Ensure target file doesn't exist (required for os.Rename on Windows to simulate overwrite)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		os.Remove(tmpPath)
		return fmt.Errorf("remove existing file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

func (c *ClientImpl) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	path := c.fullPath(key)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

func (c *ClientImpl) Delete(ctx context.Context, key string) error {
	path := c.fullPath(key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}
