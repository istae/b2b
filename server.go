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
	LengthID   = 16
	LengthKey  = 32
)

var errBadHandShake = errors.New("bad handshake")

type HandleFunc func(*Stream)

type b2b struct {
	host      string
	port      string
	peerID    string
	protocols map[string]HandleFunc        // protocol name to handleFunc
	conns     map[string]*secureReadWriter // peerID to Conn
	streams   map[string]*Stream           // peerID + steamID to Stream

	key *asymmetric
}

func New(host, port string) (*b2b, error) {
	s := &b2b{
		host:      host,
		port:      port,
		protocols: make(map[string]HandleFunc),
		conns:     make(map[string]*secureReadWriter),
		streams:   make(map[string]*Stream),
		peerID:    RandomID(),
	}

	key, err := GenerateAsymmetric()
	if err != nil {
		return nil, err
	}

	s.key = key

	return s, nil
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

			peerID, enc, err := s.sayHelloBack(conn)
			if err != nil {
				return
			}

			sc := NewSecureReadWriter(conn, enc)
			s.conns[peerID] = sc

			err = s.handle(sc)
			if err != nil {
				fmt.Println(err)
			}
		}(conn)
	}
}

func (b *b2b) handle(conn *secureReadWriter) error {

	for {

		select {
		case <-conn.C:
			return errors.New("write error")
		default:
		}

		var msg Msg

		err := conn.Read(&msg)
		if err != nil {
			return err
		}

		r, s, new := b.stream(conn, msg.Protocol, msg.PeerID, msg.StreamID)

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

	peerID, enc, err := s.sayHello(conn)
	if err != nil {
		conn.Close()
		return
	}

	sc := NewSecureReadWriter(conn, enc)
	s.conns[peerID] = sc

	go func() {
		s.handle(sc)
	}()

	return
}

func (s *b2b) Disconnect(peerID string) error {

	if c, ok := s.conns[peerID]; ok {
		return c.conn.Close()
	}

	return nil
}

func (b *b2b) NewStream(protocol, peerID string) (*Stream, error) {

	conn, ok := b.conns[peerID]
	if !ok {
		return nil, errors.New("connection does not exist")
	}

	streamID := RandomID()

	_, s, _ := b.stream(conn, protocol, peerID, streamID)

	return s, nil
}

func RandomID() string {
	id := make([]byte, LengthID)
	_, _ = rand.Read(id)
	return hex.EncodeToString(id)
}

func RandomKey() []byte {
	id := make([]byte, LengthKey)
	_, _ = rand.Read(id)
	return id
}

func (s *b2b) sayHello(c net.Conn) (peerID string, symmetricKey *symmetric, err error) {

	conn := insecureReadWriter{conn: c}

	var (
		msg = &Msg{
			Protocol: helloProcol,
			StreamID: RandomID(),
			PeerID:   s.peerID,
			Data:     s.key.pub,
		}
	)

	// send public key
	err = conn.Write(msg)
	if err != nil {
		return
	}

	// receive symmetric key
	err = conn.Read(msg)
	if err != nil {
		return
	}
	key, err := s.key.Decrypt(msg.Data)
	if err != nil {
		return
	}

	symmetricKey, err = NewSymmetricKey(key)
	peerID = msg.PeerID

	return
}

func (s *b2b) sayHelloBack(c net.Conn) (peerID string, symmetricKey *symmetric, err error) {

	conn := insecureReadWriter{conn: c}

	msg := &Msg{}

	// get public key
	err = conn.Read(msg)
	if err != nil {
		return
	}
	peerID = msg.PeerID
	pubKey := msg.Data

	// send symmetric key
	key := RandomKey()
	msg.Data, err = Encrypt(pubKey, key)
	if err != nil {
		return
	}

	msg.PeerID = s.peerID
	err = conn.Write(msg)
	if err != nil {
		return
	}

	symmetricKey, err = NewSymmetricKey(key)

	return
}
