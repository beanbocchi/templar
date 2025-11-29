package progressr

import (
	"io"
	"sync/atomic"
)

type Reader struct {
	io.Reader
	total   int64
	current atomic.Int64
}

func NewReader(reader io.Reader, total int64) *Reader {
	return &Reader{
		Reader: reader,
		total:  total,
	}
}

func (p *Reader) Read(b []byte) (int, error) {
	n, err := p.Reader.Read(b)
	p.current.Add(int64(n))
	return n, err
}

func (p *Reader) Progress() float64 {
	if p.total <= 0 {
		return 0
	}
	return float64(p.current.Load()) / float64(p.total)
}
