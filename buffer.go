package b2b

import "io"

type buffer struct {
	r io.Reader
	b []byte
	i int
}

func NewBuffer(r io.Reader) *buffer {
	return &buffer{r: r}
}

func (b *buffer) Read(length int) ([]byte, error) {

	contains := len(b.b) - b.i

	if length > contains {
		rBuf := make([]byte, length-contains)
		n, err := b.r.Read(rBuf)
		if err != nil {
			return nil, err
		}

		start := b.i
		b.i += contains + n
		b.b = append(b.b, rBuf[:n]...)

		return b.b[start:b.i], nil
	}

	start := b.i
	b.i += length

	return b.b[start:b.i], nil
}

func (b *buffer) Rewind(length int) {

	if b.i > length {
		b.i -= length
	}

	b.i = 0
}
