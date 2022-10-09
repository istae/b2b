package b2b

import (
	"errors"
	"time"
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

	maxInactive    *time.Timer
	maxInactiveDur time.Duration
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

	maxInactive := time.NewTimer(b.options.StreamMaxInactive)

	s := &Stream{
		w:              w,
		r:              make(chan []byte, 1024),
		c:              NewOnce(),
		p:              NewOnce(),
		protocol:       protocol,
		peerID:         peerID,
		streamID:       streamID,
		baseID:         b.peerID,
		maxInactive:    maxInactive,
		maxInactiveDur: b.options.StreamMaxInactive,
	}

	s.cleanUp = func() {
		b.mtx.Lock()
		delete(b.streams, id)
		b.mtx.Unlock()
		maxInactive.Stop()
	}

	go func() {
		<-s.maxInactive.C
		_ = s.Close()
	}()

	b.streams[id] = s

	return s.r, s, true
}

func (s *Stream) Write(b []byte) error {

	defer s.maxInactive.Reset(s.maxInactiveDur)

	select {
	case <-s.p.C:
		return errPeerClosed
	case <-s.c.C:
		return errStreamClosed
	default:
	}

	return s.w.Write(&Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Data: b})
}

func (s *Stream) Read([]byte) (int, error) {

	defer s.maxInactive.Reset(s.maxInactiveDur)

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
