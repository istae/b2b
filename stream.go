package b2b

import (
	"errors"
	"io"
)

type Stream struct {
	w io.Writer
	r chan []byte
	c chan struct{}
	p chan struct{}

	writeErr chan struct{}

	cleanUp func()

	streamID string
	protocol string
	peerID   string

	baseID string
}

var errStreamClosed = errors.New("stream closed by peer")
var errClosedStream = errors.New("closed stream")

func (b *b2b) stream(w io.Writer, writeErr chan struct{}, protocol, peerID, streamID string) (chan []byte, *Stream, bool) {

	id := peerID + streamID

	if s, ok := b.streams[id]; ok {
		return s.r, s, false
	}

	s := &Stream{
		w:        w,
		r:        make(chan []byte, 1024),
		c:        make(chan struct{}),
		p:        make(chan struct{}),
		writeErr: writeErr,
		cleanUp: func() {
			delete(b.streams, id)
		},
		protocol: protocol,
		peerID:   peerID,
		streamID: streamID,
		baseID:   b.peerID,
	}

	b.streams[id] = s

	return s.r, s, true
}

func (s *Stream) Write(b []byte) error {
	select {
	case <-s.p:
		return errStreamClosed
	case <-s.c:
		return errClosedStream
	default:
	}

	msg := Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Data: b}
	b, err := msg.Marshall()
	if err != nil {
		return err
	}

	err = s.write(b)
	return err
}

func (s *Stream) Read() ([]byte, error) {
	select {
	case b := <-s.r:
		return b, nil
	case <-s.p:
		return nil, errStreamClosed
	case <-s.c:
		return nil, errClosedStream
	}
}

func (s *Stream) closedByPeer() {
	close(s.p)
	s.cleanUp()
}

func (s *Stream) write(b []byte) error {

	_, err := s.w.Write(b)
	if err != nil {
		close(s.writeErr)
	}

	return err
}

func (s *Stream) Close() error {

	defer s.cleanUp()
	defer close(s.c)

	msg := Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Status: StatusClose}
	b, _ := msg.Marshall()
	return s.write(b)
}
