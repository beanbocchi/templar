package locker

import (
	"io"
	"sync"
)

// LockedReadCloser wraps a file and releases the read lock on close.
type ReadCloser struct {
	io.ReadCloser
	Lock *sync.RWMutex
}

func (l *ReadCloser) Close() error {
	err := l.ReadCloser.Close()
	l.Lock.RUnlock()
	return err
}

func NewReadCloser(r io.ReadCloser, lock *sync.RWMutex) *ReadCloser {
	return &ReadCloser{ReadCloser: r, Lock: lock}
}
