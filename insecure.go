package b2b

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"net"
)

type insecureReadWriter struct {
	conn net.Conn
}

func (s *insecureReadWriter) Write(msg encoding.BinaryMarshaler) error {

	b, err := msg.MarshalBinary()
	if err != nil {
		return err
	}

	l := make([]byte, 8)
	binary.BigEndian.PutUint32(l, uint32(len(b)))

	fmt.Println("Marshall", string(b))

	_, err = s.conn.Write(append(l, b...))
	return err
}

func (s *insecureReadWriter) Read(msg encoding.BinaryUnmarshaler) (err error) {

	buf := make([]byte, 8)
	_, err = s.conn.Read(buf)
	if err != nil {
		return
	}

	l := binary.BigEndian.Uint32(buf)
	buf = make([]byte, int(l))
	_, err = s.conn.Read(buf)
	if err != nil {
		return
	}

	fmt.Println("Unmarshall", string(buf))

	return msg.UnmarshalBinary(buf)
}
