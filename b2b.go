package b2b

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	mathRand "math/rand"
	"net"
	"sync"
	"time"
)

const (
	maxProcotolLength = 1024
	helloProcol       = "/b2b/hello/1.0.0"
)

const (
	testProcol = "test-protocol"
	LengthID   = 16
	LengthKey  = 32
)

var errPeerIDVerification = errors.New("peerID verification")

type HandleFunc func(*Stream)

type B2BOptions struct {
	StreamMaxInactive     time.Duration
	MaxConnectionsPerPeer int
}

type b2b struct {
	peerID          string
	protocols       map[string]HandleFunc          // protocol name to handleFunc
	conns           map[string][]*secureReadWriter // peerID to Conn
	streams         map[string]*Stream             // peerID + steamID to Stream
	incomingHandler func(string) error

	mtx sync.Mutex
	key *asymmetric

	options B2BOptions
}

func NewB2B(key *asymmetric) (*b2b, error) {
	b := &b2b{
		key:       key,
		protocols: make(map[string]HandleFunc),
		conns:     make(map[string][]*secureReadWriter),
		streams:   make(map[string]*Stream),
		peerID:    hex.EncodeToString(Hash(key.pubBytes)),
	}

	return b, nil
}

func (b *b2b) SetIncomingHandler(f func(peerID string) error) {
	b.incomingHandler = f
}

func (b *b2b) SetStreamMaxInactive(t time.Duration) {
	b.options.StreamMaxInactive = t
}

func (b *b2b) AddProtocol(name string, h HandleFunc) {
	b.protocols[name] = h
}

func (b *b2b) Listen(addr string) error {

	listener, err := net.Listen("tcp", addr)
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

			peerID, s, err := b.sayHelloBack(conn)
			if err != nil {
				fmt.Println(err)
				conn.Close()
				return
			}

			srw := NewSecureReadWriter(conn, s, s)
			err = b.addConn(peerID, srw)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer srw.Close()
			defer b.removeConn(peerID, srw)

			if b.incomingHandler != nil {
				err = b.incomingHandler(peerID)
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			err = b.handle(srw)
			if err != nil {
				fmt.Println(err)
			}
		}(conn)
	}
}

func (b *b2b) handle(conn *secureReadWriter) error {

	var streams []*Stream
	defer func() {
		for _, s := range streams {
			s.Close()
		}
	}()

	for {

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
			streams = append(streams, s)
			if protocolHandle, ok := b.protocols[msg.Protocol]; ok {
				go protocolHandle(s)
			}
		}

		// write to a buffered channel to preserve read order
		select {
		case r <- msg.Data:
		default:
			return errors.New("reached max read buffer")
		}
	}
}

func (b *b2b) Connect(addr string) (peerID string, err error) {

	var conn net.Conn

	conn, err = net.Dial("tcp", addr)
	if err != nil {
		return
	}

	peerID, s, err := b.sayHello(conn)
	if err != nil {
		conn.Close()
		return
	}

	srw := NewSecureReadWriter(conn, s, s)
	err = b.addConn(peerID, srw)
	if err != nil {
		srw.Close()
		return
	}

	go func() {
		defer srw.Close()
		defer b.removeConn(peerID, srw)
		err = b.handle(srw)
		if err != nil {
			fmt.Println(err)
		}
	}()

	return
}

func (b *b2b) Disconnect(peerID string) error {

	b.mtx.Lock()
	defer b.mtx.Unlock()

	for _, s := range b.conns[peerID] {
		s.Close()
	}
	delete(b.conns, peerID)

	return nil
}

func (b *b2b) NewStream(protocol, peerID string) (*Stream, error) {

	conn, ok := b.getConn(peerID)
	if !ok {
		return nil, errors.New("connection does not exist")
	}

	_, s, _ := b.stream(conn, protocol, peerID, randomID())

	return s, nil
}

func randomID() string {
	id := make([]byte, LengthID)
	_, _ = rand.Read(id)
	return hex.EncodeToString(id)
}

func (b *b2b) addConn(peerID string, srw *secureReadWriter) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.options.MaxConnectionsPerPeer > 0 && len(b.conns[peerID]) >= b.options.MaxConnectionsPerPeer {
		return errors.New("max connections per peer reached")
	}

	b.conns[peerID] = append(b.conns[peerID], srw)

	return nil
}

func (b *b2b) removeConn(peerID string, srw *secureReadWriter) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	conns := b.conns[peerID]

	for i, c := range conns {
		if c == srw {
			b.conns[peerID] = append(conns[:i], conns[i+1:]...)
			return
		}
	}
}

func (b *b2b) getConn(peerID string) (*secureReadWriter, bool) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	srw := b.conns[peerID]

	if len(srw) > 0 {
		return srw[mathRand.Intn(len(srw))], true
	}

	return nil, false
}
