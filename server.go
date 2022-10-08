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
	peerID    string
	protocols map[string]HandleFunc // protocol name to protocol
	conns     map[string]net.Conn   // peerID to conn
	streams   map[string]*Stream
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

	quitErr := make(chan struct{})

	for {

		select {
		case <-quitErr:
			return errors.New("write error")
		default:
		}

		var msg Msg

		err := msg.Unmarshall(conn)
		if err != nil {
			continue
		}

		r, s, new := b.stream(conn, quitErr, msg.Protocol, msg.PeerID, msg.StreamID)

		if msg.Status == StatusClose {
			s.closedByPeer()
			continue
		}

		if new {
			if protocolHandle, ok := b.protocols[msg.Protocol]; ok {
				go protocolHandle(s)
			}
		}

		select {
		case r <- msg.Data:
		default:
			return errors.New("reached max read buffer")
		}
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

func (s *b2b) Disconnect(peerID string) error {

	if conn, ok := s.conns[peerID]; ok {
		return conn.Close()
	}

	return nil
}

func (b *b2b) NewStream(protocol, peerID string) (*Stream, error) {

	conn, ok := b.conns[peerID]
	if !ok {
		return nil, errors.New("connection does not exist")
	}

	streamID := randomID()

	_, s, _ := b.stream(conn, nil, protocol, peerID, streamID)

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
