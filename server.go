package b2b

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
)

/*
/b2b/[protocol]/[version]
[streamID][length][payload]
*/

const (
	maxProcotolLength = 1024
	helloProcol       = "/b2b/hello/1.0.0"
)

const (
	testProcol = "test-protocol"
)

var errBadHandShake = errors.New("bad handshake")

type HandleFunc func(*Stream)

type b2b struct {
	host      string
	port      string
	protocols map[string]HandleFunc // protocol name to protocol
	conns     map[string]net.Conn   // peerID to conn
	streams   map[string]*Stream

	peerID string
}

func New(host, port string) *b2b {
	s := &b2b{
		host:      host,
		port:      port,
		protocols: make(map[string]HandleFunc),
		conns:     make(map[string]net.Conn),
		streams:   make(map[string]*Stream),
		peerID:    randomID(),
	}
	return s
}

func (b *b2b) AddProcol(name string, h HandleFunc) {
	b.protocols[name] = h
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

		go func(conn net.Conn) {

			defer conn.Close()

			peerID, err := s.sayHelloBack(conn)
			if err != nil {
				return
			}

			s.conns[peerID] = conn

			err = s.handle(conn)
			if err != nil {
				fmt.Println(err)
			}
		}(conn)
	}
}

func (b *b2b) handle(conn net.Conn) error {

	for {
		var msg Msg

		err := msg.Unmarshall(conn)
		if err != nil {
			continue
		}

		w, r, c, s, new := b.stream(msg.Protocol, msg.PeerID, msg.StreamID)

		if new {
			if protocolHandle, ok := b.protocols[msg.Protocol]; ok {
				go protocolHandle(s)
				go func(msg Msg, w chan []byte, c chan struct{}, s *Stream) {
					for {
						select {
						case <-c:
							fmt.Println("closed")
							return
						case data := <-w:
							msg := Msg{Protocol: msg.Protocol, StreamID: msg.StreamID, PeerID: b.peerID, Data: data}
							b, _ := msg.Marshall()
							_, err := conn.Write(b)
							if err != nil {
								fmt.Println("err on conn.Write")
								s.close()
								return
							}
						}
					}
				}(msg, w, c, s)
			}
		}

		go func(msg Msg, r chan []byte, c chan struct{}) {
			for {
				select {
				case <-c:
					fmt.Println("closed")
					return
				case r <- msg.Data:
				}
			}
		}(msg, r, c)
	}
}

func (s *b2b) Connect(addr string) (peerID string, err error) {

	var conn net.Conn

	conn, err = net.Dial("tcp", addr)
	if err != nil {
		return
	}

	peerID, err = s.sayHello(conn)
	if err != nil {
		conn.Close()
		return
	}

	go func() {
		s.handle(conn)
		conn.Close()
	}()

	s.conns[peerID] = conn
	return
}

func (b *b2b) NewStream(protocol, peerID string) (*Stream, error) {

	conn, ok := b.conns[peerID]
	if !ok {
		return nil, errors.New("connection does not exist")
	}

	streamID := randomID()

	w, _, c, s, _ := b.stream(protocol, peerID, streamID)

	go func() {
		for {
			select {
			case <-c:
				return
			case data := <-w:
				msg := Msg{Protocol: protocol, StreamID: streamID, PeerID: b.peerID, Data: data}
				b, _ := msg.Marshall()
				_, err := conn.Write(b)
				if err != nil {
					fmt.Println("err on conn.Write")
					s.close()
					return
				}
			}
		}
	}()

	return s, nil
}

func randomID() string {
	id := make([]byte, 4)
	_, _ = rand.Read(id)
	return hex.EncodeToString(id)
}

func (s *b2b) sayHello(conn net.Conn) (peerID string, err error) {

	var (
		msg = Msg{
			Protocol: helloProcol,
			StreamID: randomID(),
			PeerID:   s.peerID,
		}
		b []byte
	)

	b, err = msg.Marshall()
	if err != nil {
		return
	}

	_, err = conn.Write(b)
	if err != nil {
		return
	}

	err = msg.Unmarshall(conn)
	if err != nil {
		return
	}

	if msg.Protocol != helloProcol {
		return
	}

	peerID = msg.PeerID

	return
}

func (s *b2b) sayHelloBack(conn net.Conn) (peerID string, err error) {

	var msg Msg
	var b []byte

	err = msg.Unmarshall(conn)
	if err != nil {
		return
	}

	if msg.Protocol != helloProcol {
		err = errBadHandShake
		return
	}

	peerID = msg.PeerID
	msg.PeerID = s.peerID

	b, err = msg.Marshall()
	if err != nil {
		return
	}

	_, err = conn.Write(b)

	return
}

// func readProtocol(buf *buffer) (string, error) {

// 	b, err := buf.Read(maxProcotolLength)
// 	if err != nil {
// 		return "", err
// 	}

// 	indx := strings.Index(string(b), "\n")

// 	return string(b[:indx]), nil
// }

// func (s *b2b) addConn(conn net.Conn) {

// 	r := make([]byte, 1024)

// 	go func() {
// 		for {
// 			_, err := conn.Read(r)
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 		}
// 	}()

// 	go func() {
// 		for {
// 			conn.Read(r)
// 		}
// 	}()
// }

// a stream is a protol specific read and write

/*

type stream struct {

	w io.Writer
	r io.Reader

}


func NewStream() *stream {

}

func (s *b2b) handle(conn net.Conn) error {

	var (
		buffer = NewBuffer(conn)
		err    error
	)

	protocol, err := readProtocol(buffer)
	if err != nil {
		return err
	}

	buffer.Rewind(maxProcotolLength - len(protocol))

	if protocol != helloProcol {
		return errors.New("not hello protocol")
	}

	_, err = conn.Write([]byte("/b2b/hello/1.0.0\n"))
	if err != nil {
		return err
	}

	s.conns[testPeerID] = conn

	for {

	}

	return err
}



*/
