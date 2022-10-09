package b2b

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

const (
	StatusOk = iota
	StatusClose
)

type Msg struct {
	Protocol string
	StreamID string
	PeerID   string
	Status   int
	Data     []byte
}

func (m *Msg) MarshalBinary() ([]byte, error) {
	return json.Marshal(&m)
}

func (m *Msg) UnmarshalBinary(b []byte) error {
	return json.Unmarshal(b, m)
}

func DecodeLength(r io.Reader) (uint64, error) {

	buf := make([]byte, 8)
	n, err := r.Read(buf)
	if err != nil {
		return 0, err
	}

	if n != 8 {
		return 0, errors.New("read bytes fewer than size of uint64")
	}

	return binary.BigEndian.Uint64(buf), nil
}

func EncodeLength(l uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, l)
	return b
}
