package b2b

import (
	"errors"
)

type Stream struct {
	w *secureReadWriter
	r chan []byte
	c chan struct{}
	p chan struct{}

	cleanUp func()

	streamID string
	protocol string
	peerID   string

	baseID string
}

var errStreamClosed = errors.New("stream closed by peer")
var errClosedStream = errors.New("closed stream")

func (b *b2b) stream(w *secureReadWriter, protocol, peerID, streamID string) (chan []byte, *Stream, bool) {

	id := peerID + streamID

	if s, ok := b.streams[id]; ok {
		return s.r, s, false
	}

	s := &Stream{
		w:        w,
		r:        make(chan []byte, 1024),
		c:        make(chan struct{}),
		p:        make(chan struct{}),
		protocol: protocol,
		peerID:   peerID,
		streamID: streamID,
		baseID:   b.peerID,
		cleanUp: func() {
			delete(b.streams, id)
		},
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

	msg := &Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Data: b}
	return s.write(msg)
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

func (s *Stream) write(msg *Msg) error {

	err := s.w.Write(msg)
	if err != nil {
		s.w.Close()
		return err
	}

	return nil
}

func (s *Stream) Close() error {

	defer s.cleanUp()
	defer close(s.c)

	msg := &Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Status: StatusClose}
	return s.w.Write(msg)
}
