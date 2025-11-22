package progress

import "io"

type Reader struct {
	io.Reader
	total   int64
	current int64
}

func (p *Reader) Read(b []byte) (int, error) {
	n, err := p.Reader.Read(b)
	p.current += int64(n)
	return n, err
}

func NewReader(reader io.Reader, total int64) *Reader {
	return &Reader{
		Reader:  reader,
		total:   total,
		current: 0,
	}
}

func (p *Reader) Progress() float64 {
	return float64(p.current) / float64(p.total)
}
