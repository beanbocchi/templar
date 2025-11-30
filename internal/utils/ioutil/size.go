package ioutil

import (
	"io"
)

type SizeReader struct {
	io.Reader
	Size int64
}

func NewSizeReader(reader io.Reader) *SizeReader {
	return &SizeReader{Reader: reader}
}

func (s *SizeReader) Read(p []byte) (n int, err error) {
	n, err = s.Reader.Read(p)
	s.Size += int64(n)
	return n, err
}
