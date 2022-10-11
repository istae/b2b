package b2b

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"net"
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
}

func NewSecureReadWriter(conn net.Conn, enc Encrypter, dec Decrypter) *secureReadWriter {
	srw := &secureReadWriter{
		conn: conn,
		enc:  enc,
		dec:  dec,
	}

	return srw
}

func (s *secureReadWriter) Write(m encoding.BinaryMarshaler) error {

	b, err := m.MarshalBinary()
	if err != nil {
		return err
	}

	fmt.Println("Marshall", string(b))

	data, err := s.enc.Encrypt(b)
	if err != nil {
		return err
	}

	l := make([]byte, 8)
	binary.BigEndian.PutUint32(l, uint32(len(data)))

	// returned write length is not the actual length of the data
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

	data, err := s.dec.Decrypt(b)
	if err != nil {
		return
	}

	fmt.Println("Unmarshall", string(data))

	return m.UnmarshalBinary(data)
}

func (s *secureReadWriter) Close() error {
	return s.conn.Close()
}
