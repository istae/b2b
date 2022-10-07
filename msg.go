package b2b

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
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

func (m *Msg) Marshall() ([]byte, error) {
	b, err := json.Marshal(&m)
	if err != nil {
		return nil, err
	}

	fmt.Println("Marshall", string(b))

	// fmt.Println(string(b))

	lengthBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(lengthBuf, uint64(len(b)))

	// fmt.Println("Marshall", "length", len(b))

	return append(lengthBuf, b...), nil

}

func (m *Msg) Unmarshall(r io.Reader) error {

	buf := make([]byte, 8)
	_, err := r.Read(buf)
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint64(buf)

	// fmt.Println("Unmarshall", "length", length)

	buf = make([]byte, int(length))

	_, err = r.Read(buf)
	if err != nil {
		return err
	}

	fmt.Println("Unmarshall", string(buf))

	json.Unmarshal(buf, &m)

	return json.Unmarshal(buf, m)
}
