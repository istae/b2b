package b2b

import (
	"errors"
	"fmt"
)

type Stream struct {
	w chan []byte
	r chan []byte
	c chan struct{}

	closeFunc func()

	streamID string
	protocol string
	peerID   string
}

var errStreamClosed = errors.New("stream closed")

func (b *b2b) stream(protocol, peerID, streamID string) (chan []byte, chan []byte, chan struct{}, *Stream, bool) {

	id := peerID + streamID

	if s, ok := b.streams[id]; ok {
		return s.w, s.r, s.c, s, false
	}

	s := &Stream{
		w: make(chan []byte),
		r: make(chan []byte),
		c: make(chan struct{}),
		closeFunc: func() {
			delete(b.streams, id)
		},
		protocol: "",
		peerID:   peerID,
		streamID: streamID,
	}

	b.streams[id] = s

	return s.w, s.r, s.c, s, true
}

func (s *Stream) Write(b []byte) error {
	fmt.Println("write start")
	select {
	case s.w <- b:
		fmt.Println("write finished")
		return nil
	case <-s.c:
		fmt.Println("write closed")
		return errStreamClosed
	}
}

func (s *Stream) Read() ([]byte, error) {
	fmt.Println("read start")
	select {
	case b := <-s.r:
		fmt.Println("read finished")
		return b, nil
	case <-s.c:
		fmt.Println("read closed")
		return nil, errStreamClosed
	}
}

func (s *Stream) close() {
	s.closeFunc()
	close(s.c)
}

func (s *Stream) Close() error {

	msg := Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.peerID, Status: StatusClose}
	b, err := msg.Marshall()
	if err != nil {
		return err
	}

	select {
	case <-s.c:
		return nil
	case s.w <- b:
		s.close()
		return nil
	}
}
