package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ClientImpl struct {
	root    string
	baseURL string
	locks   sync.Map // map[string]*sync.RWMutex
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
