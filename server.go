package b2b

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

/*
/b2b/[protocol]/[version]
[streamID][length][payload]
*/

var errBadHandShake = errors.New("bad handshake")

type b2b struct {
	host      string
	port      string
	protocols map[string]protocol // protocol name to protocol
	conns     map[string]net.Conn // peerID to conn
}

type protocol struct {
	handleFunc func()
	streams    map[string]stream
}

type stream struct {
	io.ReadWriteCloser
	streamID string
}

func New(host, port string) *b2b {
	s := &b2b{
		host: host,
		port: port,
	}
	return s
}

func (s *b2b) Listen() error {

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", s.host, s.port))
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go func() {
			err = handleConnection(conn)
			if err != nil {
				conn.Close()
				fmt.Println(err)
			}
		}()
	}
}

func (s *b2b) Connect(host, port string) (net.Conn, error) {

	var (
		conn net.Conn
		err  error
	)

	conn, err = net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	_, err = conn.Write([]byte("/b2b/hello/1.0.0\n"))
	if err != nil {
		return nil, err
	}

	b := make([]byte, 5)

	fmt.Println("waiting on read")
	_, err = conn.Read(b)
	if err != nil {
		return nil, err
	}

	fmt.Println("received")
	if string(b) == "hello" {
		return conn, nil
	}

	return nil, errBadHandShake
}

func handleConnection(conn net.Conn) error {

	b := make([]byte, 5)

	_, err := conn.Read(b)
	if err != nil {
		return err
	}

	if string(b) != "hello" {
		return errBadHandShake
	}

	_, err = conn.Write([]byte("hello"))
	if err != nil {
		return err
	}

	fmt.Println("handshaken with", conn.RemoteAddr().String())

	return nil
}

func (s *b2b) addConn(conn net.Conn) {

	r := make([]byte, 1024)

	go func() {
		for {
			_, err := conn.Read(r)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	go func() {
		for {
			conn.Read(r)
		}
	}()
}

// a stream is a protol specific read and write

/*

type stream struct {

	w io.Writer
	r io.Reader

}


func NewStream() *stream {

}



*/
