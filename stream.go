package b2b

import (
	"errors"
	"io"
	"sync"
	"time"
)

type Stream struct {
	mtx   sync.Mutex
	w     *secureReadWriter // writer
	r     chan io.Reader    // buffered channel that stores stream bytes to read
	lastR io.Reader

	cleanUp  func()
	c        *Once // signals closed channel
	p        *Once // signals stream closed by peer
	streamID string
	protocol string
	peerID   string
	baseID   string

	maxInactive    *time.Timer // closes the stream after an inactive period, reset by Read and Write calls
	maxInactiveDur time.Duration
}

var errPeerClosed = errors.New("stream closed by peer")
var errStreamClosed = errors.New("stream closed")

func (b *b2b) stream(w *secureReadWriter, protocol, peerID, streamID string) (chan io.Reader, *Stream, bool) {

	b.mtx.Lock()
	defer b.mtx.Unlock()

	id := peerID + streamID

	if s, ok := b.streams[id]; ok {
		return s.r, s, false
	}

	maxInactive := time.NewTimer(b.options.StreamMaxInactive)

	s := &Stream{
		w:              w,
		r:              make(chan io.Reader, 1024),
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

func (s *Stream) Write(b []byte) (int, error) {

	defer s.maxInactive.Reset(s.maxInactiveDur)

	select {
	case <-s.p.C:
		return 0, errPeerClosed
	case <-s.c.C:
		return 0, errStreamClosed
	default:
	}

	return s.w.Write(&Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Data: b})
}

func (s *Stream) Read(b []byte) (int, error) {

	defer s.maxInactive.Reset(s.maxInactiveDur)

	s.mtx.Lock()
	if s.lastR != nil {
		n, err := s.lastR.Read(b)
		if errors.Is(err, io.EOF) {
			s.lastR = nil
		}
		s.mtx.Unlock()
		return n, err
	}
	s.mtx.Unlock()

	select {
	case r := <-s.r:
		n, err := r.Read(b)
		if !errors.Is(err, io.EOF) {
			s.mtx.Lock()
			s.lastR = r
			s.mtx.Unlock()
		}
		return n, err
	case <-s.p.C:
		return 0, errPeerClosed
	case <-s.c.C:
		return 0, errStreamClosed
	}
}

func (s *Stream) Close() error {
	s.cleanUp()
	s.c.Done()
	s.w.Write(&Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Status: StatusClose})
	return nil
}

func (s *Stream) closedByPeer() {
	s.cleanUp()
	s.p.Done()
}
