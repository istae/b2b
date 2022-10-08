package b2b

import (
	"encoding/json"
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
