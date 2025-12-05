package objectstore

import (
	"context"
	"io"
)

type Client interface {
	// Start a multipart upload session
	CreateMultipart(ctx context.Context, key string) (uploadID string, err error)

	// Upload a single part
	UploadPart(
		ctx context.Context,
		key string,
		uploadID string,
		partNumber int,
		content io.Reader,
	) error

	// Finish the upload by combining uploaded parts
	CompleteMultipart(
		ctx context.Context,
		key string,
		uploadID string,
	) error

	// Cancel and cleanup partial data
	AbortMultipart(
		ctx context.Context,
		key string,
		uploadID string,
	) error

	Upload(ctx context.Context, key string, content io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
