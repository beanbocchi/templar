package sync

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockClient is a mock implementation of objectstore.Client for testing
type mockClient struct {
	mu            sync.Mutex
	uploads       map[string][]byte
	downloads     map[string][]byte
	deletes       []string
	uploadErr     error
	downloadErr   error
	deleteErr     error
	uploadDelay   time.Duration
	downloadDelay time.Duration
}

func newMockClient() *mockClient {
	return &mockClient{
		uploads:   make(map[string][]byte),
		downloads: make(map[string][]byte),
		deletes:   make([]string, 0),
	}
}

func (m *mockClient) Upload(ctx context.Context, key string, content io.Reader) error {
	if m.uploadDelay > 0 {
		time.Sleep(m.uploadDelay)
	}
	if m.uploadErr != nil {
		return m.uploadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	m.uploads[key] = data
	return nil
}

func (m *mockClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.downloadDelay > 0 {
		time.Sleep(m.downloadDelay)
	}
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.downloads[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockClient) Delete(ctx context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletes = append(m.deletes, key)
	delete(m.uploads, key)
	delete(m.downloads, key)
	return nil
}

func TestNewSyncClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockClient()
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("expected client to be non-nil")
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		_, err := NewSyncClient(SyncConfig{Client: nil})
		if err == nil {
			t.Fatal("expected error for nil client")
		}
		if !strings.Contains(err.Error(), "client is required") {
			t.Errorf("expected error about client being required, got %v", err)
		}
	})
}

func TestSyncUpload(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock := newMockClient()
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		key := "test.txt"
		content := "test content"
		err = client.Upload(ctx, key, strings.NewReader(content))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(mock.uploads) == 0 {
			t.Error("expected upload to succeed")
		}
		if string(mock.uploads[key]) != content {
			t.Errorf("expected content %q, got %q", content, string(mock.uploads[key]))
		}
	})

	t.Run("concurrent uploads to same key are serialized", func(t *testing.T) {
		mock := newMockClient()
		mock.uploadDelay = 10 * time.Millisecond
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		key := "concurrent.txt"
		var wg sync.WaitGroup
		errors := make([]error, 3)

		// Start 3 concurrent uploads to the same key
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				content := strings.NewReader("content" + string(rune('0'+idx)))
				errors[idx] = client.Upload(ctx, key, content)
			}(i)
		}

		wg.Wait()

		// All should succeed
		for i, err := range errors {
			if err != nil {
				t.Errorf("upload %d failed: %v", i, err)
			}
		}

		// Only one upload should have succeeded (last one wins)
		if len(mock.uploads) != 1 {
			t.Errorf("expected 1 upload, got %d", len(mock.uploads))
		}
	})

	t.Run("concurrent uploads to different keys are parallel", func(t *testing.T) {
		mock := newMockClient()
		mock.uploadDelay = 10 * time.Millisecond
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		var wg sync.WaitGroup
		keys := []string{"key1.txt", "key2.txt", "key3.txt"}

		start := time.Now()
		for _, key := range keys {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				client.Upload(ctx, k, strings.NewReader("content"))
			}(key)
		}
		wg.Wait()
		duration := time.Since(start)

		// With parallel execution, should take roughly the same time as one upload
		// (not 3x the time)
		if duration > 30*time.Millisecond {
			t.Errorf("expected parallel execution, took %v", duration)
		}

		// All keys should be uploaded
		if len(mock.uploads) != 3 {
			t.Errorf("expected 3 uploads, got %d", len(mock.uploads))
		}
	})
}

func TestSyncDownload(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock := newMockClient()
		mock.downloads["test.txt"] = []byte("test content")
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		reader, err := client.Download(ctx, "test.txt")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read: %v", err)
		}
		if string(data) != "test content" {
			t.Errorf("expected %q, got %q", "test content", string(data))
		}
	})

	t.Run("concurrent downloads to same key are parallel", func(t *testing.T) {
		mock := newMockClient()
		mock.downloads["test.txt"] = []byte("test content")
		mock.downloadDelay = 10 * time.Millisecond
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		var wg sync.WaitGroup
		results := make([][]byte, 3)

		start := time.Now()
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				reader, err := client.Download(ctx, "test.txt")
				if err != nil {
					return
				}
				defer reader.Close()
				data, _ := io.ReadAll(reader)
				results[idx] = data
			}(i)
		}
		wg.Wait()
		duration := time.Since(start)

		// With parallel reads, should take roughly the same time as one download
		if duration > 30*time.Millisecond {
			t.Errorf("expected parallel reads, took %v", duration)
		}

		// All should have read the same content
		for i, data := range results {
			if string(data) != "test content" {
				t.Errorf("download %d: expected %q, got %q", i, "test content", string(data))
			}
		}
	})

	t.Run("read lock released on close", func(t *testing.T) {
		mock := newMockClient()
		mock.downloads["test.txt"] = []byte("test content")
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		reader, err := client.Download(ctx, "test.txt")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Close should release the lock
		err = reader.Close()
		if err != nil {
			t.Fatalf("close failed: %v", err)
		}

		// After close, we should be able to upload (write lock) immediately
		start := time.Now()
		err = client.Upload(ctx, "test.txt", strings.NewReader("new content"))
		duration := time.Since(start)
		if err != nil {
			t.Fatalf("upload failed: %v", err)
		}
		// Should not be blocked
		if duration > 50*time.Millisecond {
			t.Errorf("expected no blocking after close, took %v", duration)
		}
	})

	t.Run("download error releases lock", func(t *testing.T) {
		mock := newMockClient()
		mock.downloadErr = errors.New("download failed")
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.Download(ctx, "test.txt")
		if err == nil {
			t.Fatal("expected error")
		}

		// Lock should be released, so we can upload immediately
		err = client.Upload(ctx, "test.txt", strings.NewReader("content"))
		if err != nil {
			t.Fatalf("expected upload to succeed after download error, got %v", err)
		}
	})
}

func TestSyncDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock := newMockClient()
		mock.uploads["test.txt"] = []byte("content")
		mock.downloads["test.txt"] = []byte("content")
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.Delete(ctx, "test.txt")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(mock.deletes) == 0 || mock.deletes[0] != "test.txt" {
			t.Errorf("expected test.txt to be deleted")
		}
	})

	t.Run("concurrent delete and upload are serialized", func(t *testing.T) {
		mock := newMockClient()
		mock.uploadDelay = 10 * time.Millisecond
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		key := "concurrent.txt"
		var deleteErr, uploadErr error
		var wg sync.WaitGroup

		wg.Add(2)
		go func() {
			defer wg.Done()
			deleteErr = client.Delete(ctx, key)
		}()
		go func() {
			defer wg.Done()
			uploadErr = client.Upload(ctx, key, strings.NewReader("content"))
		}()

		wg.Wait()

		// Both should succeed (one will wait for the other)
		if deleteErr != nil {
			t.Errorf("delete failed: %v", deleteErr)
		}
		if uploadErr != nil {
			t.Errorf("upload failed: %v", uploadErr)
		}
	})
}

func TestSyncReadWriteLocking(t *testing.T) {
	ctx := context.Background()

	t.Run("write blocks reads", func(t *testing.T) {
		mock := newMockClient()
		mock.uploadDelay = 50 * time.Millisecond
		client, err := NewSyncClient(SyncConfig{Client: mock})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		key := "lock.txt"
		readStarted := make(chan bool)
		readDone := make(chan bool)

		// Start upload (write lock)
		go func() {
			client.Upload(ctx, key, strings.NewReader("content"))
		}()

		// Small delay to ensure upload starts first
		time.Sleep(5 * time.Millisecond)

		// Start download (read lock) - should wait for write to complete
		go func() {
			readStarted <- true
			reader, _ := client.Download(ctx, key)
			if reader != nil {
				reader.Close()
			}
			readDone <- true
		}()

		// Wait for read to start
		<-readStarted

		// Read should be blocked, so it shouldn't complete immediately
		select {
		case <-readDone:
			t.Error("read should be blocked by write")
		case <-time.After(20 * time.Millisecond):
			// Good, read is blocked
		}

		// Wait for write to complete
		time.Sleep(60 * time.Millisecond)

		// Now read should complete
		select {
		case <-readDone:
			// Good
		case <-time.After(100 * time.Millisecond):
			t.Error("read should complete after write")
		}
	})
}
