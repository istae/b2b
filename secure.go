package b2b

import (
	"encoding"
	"encoding/binary"
	"net"
	"sync"
)

type secureReadWriter struct {
	conn net.Conn
	enc  *symmetric
	once sync.Once
	C    chan struct{}
}

func NewSecureReadWriter(conn net.Conn, enc *symmetric) *secureReadWriter {
	return &secureReadWriter{
		C:    make(chan struct{}),
		conn: conn,
		enc:  enc,
	}
}

func (s *secureReadWriter) Write(m encoding.BinaryMarshaler) error {

	b, err := m.MarshalBinary()
	if err != nil {
		return err
	}

	data, err := s.enc.Encrypt(b)
	if err != nil {
		return err
	}

	l := make([]byte, 8)
	binary.BigEndian.PutUint32(l, uint32(len(data)))

	_, err = s.conn.Write(append(l, data...))
	return err
}

func (s *secureReadWriter) Read(m encoding.BinaryUnmarshaler) (err error) {

	b := make([]byte, 8)
	_, err = s.conn.Read(b)
	if err != nil {
		return
	}

	l := binary.BigEndian.Uint32(b)
	b = make([]byte, int(l))
	_, err = s.conn.Read(b)
	if err != nil {
		return
	}

	data, err := s.enc.Decrypt(b)
	if err != nil {
		return
	}

	return m.UnmarshalBinary(data)
}

func (s *secureReadWriter) Close() error {
	s.once.Do(func() { close(s.C) })
	return s.conn.Close()
}
