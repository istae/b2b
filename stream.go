package b2b

import (
	"errors"
)

type Stream struct {
	w *secureReadWriter
	r chan []byte

	cleanUp  func()
	c        *Once
	p        *Once
	streamID string
	protocol string
	peerID   string
	baseID   string
}

var errPeerClosed = errors.New("stream closed by peer")
var errStreamClosed = errors.New("stream closed")

func (b *b2b) stream(w *secureReadWriter, protocol, peerID, streamID string) (chan []byte, *Stream, bool) {

	b.mtx.Lock()
	defer b.mtx.Unlock()

	id := peerID + streamID

	if s, ok := b.streams[id]; ok {
		return s.r, s, false
	}

	s := &Stream{
		w:        w,
		r:        make(chan []byte, 1024),
		c:        NewOnce(),
		p:        NewOnce(),
		protocol: protocol,
		peerID:   peerID,
		streamID: streamID,
		baseID:   b.peerID,
		cleanUp: func() {
			b.mtx.Lock()
			delete(b.streams, id)
			b.mtx.Unlock()
		},
	}

	b.streams[id] = s

	return s.r, s, true
}

func (s *Stream) Write(b []byte) error {
	select {
	case <-s.p.C:
		return errPeerClosed
	case <-s.c.C:
		return errStreamClosed
	default:
	}

	return s.w.Write(&Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Data: b})
}

func (s *Stream) Read() ([]byte, error) {
	select {
	case b := <-s.r:
		return b, nil
	case <-s.p.C:
		return nil, errPeerClosed
	case <-s.c.C:
		return nil, errStreamClosed
	}
}

func (s *Stream) Close() error {
	s.cleanUp()
	s.c.Done()
	return s.w.Write(&Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Status: StatusClose})
}

func (s *Stream) closedByPeer() {
	s.cleanUp()
	s.p.Done()
}
