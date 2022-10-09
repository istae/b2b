package pipe_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/istae/b2b"
	"github.com/istae/b2b/pipe"
)

func TestX(t *testing.T) {

	buf := bytes.Buffer{}

	l := b2b.EncodeLength(100)
	buf.Write(l)

	data := randBytes(100)

	buf.Write(data)

	p := pipe.NewBuffer(&buf)

	test := make([]byte, 1000)
	n, err := p.Read(test)
	if err != nil {
		t.Fatal()
	}
	if n != 100 {
		t.Fatal("mismatch length")
	}

	if hex.EncodeToString(data) != hex.EncodeToString(test[:n]) {
		t.Fatal("data length")
	}

}

func randBytes(l int) []byte {

	b := make([]byte, l)
	_, _ = rand.Read(b)

	return b
}
