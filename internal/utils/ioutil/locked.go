package ioutil

import (
	"io"
	"sync"
)

// LockedReadCloser wraps a file and releases the read lock on close.
type LockedReadCloser struct {
	io.ReadCloser
	Lock *sync.RWMutex
}

func (l *LockedReadCloser) Close() error {
	err := l.ReadCloser.Close()
	l.Lock.RUnlock()
	return err
}

func NewLockedReadCloser(r io.ReadCloser, lock *sync.RWMutex) *LockedReadCloser {
	return &LockedReadCloser{ReadCloser: r, Lock: lock}
}
