package objectstore

import (
	"context"
	"io"
)

type Client interface {
	Upload(ctx context.Context, key string, content io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
