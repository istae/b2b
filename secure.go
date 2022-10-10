package b2b

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type Encrypter interface {
	Encrypt([]byte) ([]byte, error)
}

type Decrypter interface {
	Decrypt([]byte) ([]byte, error)
}

type secureReadWriter struct {
	conn net.Conn
	enc  Encrypter
	dec  Decrypter

	maxInactive    *time.Timer
	maxInactiveDur time.Duration
}

func NewSecureReadWriter(conn net.Conn, enc Encrypter, dec Decrypter, inactive time.Duration) *secureReadWriter {
	srw := &secureReadWriter{
		conn:           conn,
		enc:            enc,
		dec:            dec,
		maxInactive:    time.NewTimer(inactive),
		maxInactiveDur: inactive,
	}

	go func() {
		<-srw.maxInactive.C
		_ = srw.Close()
	}()

	return srw
}

func (s *secureReadWriter) Write(m encoding.BinaryMarshaler) (int, error) {

	defer s.maxInactive.Reset(s.maxInactiveDur)

	b, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}

	fmt.Println("Marshall", string(b))

	data, err := s.enc.Encrypt(b)
	if err != nil {
		return 0, err
	}

	l := make([]byte, 8)
	binary.BigEndian.PutUint32(l, uint32(len(data)))

	// returned write length is not the actual length of the data
	return s.conn.Write(append(l, data...))
}

func (s *secureReadWriter) Read(m encoding.BinaryUnmarshaler) (err error) {

	defer s.maxInactive.Reset(s.maxInactiveDur)

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

	data, err := s.dec.Decrypt(b)
	if err != nil {
		return
	}

	fmt.Println("Unmarshall", string(data))

	return m.UnmarshalBinary(data)
}

func (s *secureReadWriter) Close() error {
	s.maxInactive.Stop()
	return s.conn.Close()
}
