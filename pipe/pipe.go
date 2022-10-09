package pipe

import (
	"io"

	"github.com/istae/b2b"
)

type pipe struct {
	r io.Reader

	// mtx sync.Mutex

	end   uint64
	start uint64
}

func NewReader(r io.ReadWriter) *pipe {
	return &pipe{r: r}
}

func (p *pipe) Read(b []byte) (int, error) {

	// p.mtx.Lock()
	// defer p.mtx.Unlock()

	// start of section
	if p.start == p.end {
		l, err := b2b.DecodeLength(p.r)
		if err != nil {
			return 0, err
		}
		p.end = l
	}

	c := p.end - p.start

	// cannot read more than allowed by section
	if len(b) > int(c) {
		b = b[:c]
	}

	n, err := p.r.Read(b)
	if err != nil {
		return 0, err
	}

	p.start += uint64(n)

	return n, err
}

func (p *pipe) PipeWriter(l uint64) io.Writer {
	return nil
}

// func (p *pipe) Write([]byte) (int, error) {
// 	return 0, nil
// }
