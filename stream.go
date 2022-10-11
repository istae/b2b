package b2b

import (
	"errors"
	"sync"
)

type Stream struct {
	mtx sync.Mutex
	w   *secureReadWriter // writer
	r   chan []byte       // buffered channel that stores stream bytes to read

	cleanUp  func()
	c        *Once // signals closed channel
	p        *Once // signals stream closed by peer
	streamID string
	protocol string
	peerID   string
	baseID   string

	maxInactive *Timer // closes the stream after an inactive period, reset by Read and Write calls
}

var errPeerClosed = errors.New("stream closed by peer")
var errStreamClosed = errors.New("stream closed")
var errTimeout = errors.New("stream timed out")

func (b *b2b) stream(w *secureReadWriter, protocol, peerID, streamID string) (chan []byte, *Stream, bool) {

	b.mtx.Lock()
	defer b.mtx.Unlock()

	id := peerID + streamID

	if s, ok := b.streams[id]; ok {
		return s.r, s, false
	}

	s := &Stream{
		w:           w,
		r:           make(chan []byte, 1024),
		c:           NewOnce(),
		p:           NewOnce(),
		protocol:    protocol,
		peerID:      peerID,
		streamID:    streamID,
		baseID:      b.peerID,
		maxInactive: NewTimer(b.options.StreamMaxInactive),
	}

	s.cleanUp = func() {
		b.mtx.Lock()
		delete(b.streams, id)
		b.mtx.Unlock()
		s.maxInactive.Close()
	}

	go func() {
		<-s.maxInactive.C
		_ = s.Close()
	}()

	b.streams[id] = s

	return s.r, s, true
}

func (s *Stream) Write(b []byte) error {

	defer s.maxInactive.Reset()

	select {
	case <-s.maxInactive.C:
		return errTimeout
	case <-s.p.C:
		return errPeerClosed
	case <-s.c.C:
		return errStreamClosed
	default:
		return s.w.Write(&Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Data: b})
	}
}

func (s *Stream) Read() ([]byte, error) {

	defer s.maxInactive.Reset()

	if len(s.r) == 0 {
		select {
		case <-s.maxInactive.C:
			return nil, errTimeout
		case <-s.p.C:
			return nil, errPeerClosed
		case <-s.c.C:
			return nil, errStreamClosed
		case b := <-s.r:
			return b, nil
		}
	}

	return <-s.r, nil
}

func (s *Stream) Close() error {

	s.cleanUp()
	s.c.Done()

	select {
	case <-s.p.C: // if peer closed, no need to write close message
		return nil
	default:
		return s.w.Write(&Msg{Protocol: s.protocol, StreamID: s.streamID, PeerID: s.baseID, Status: StatusClose})
	}
}

func (s *Stream) closedByPeer() {
	s.cleanUp()
	s.p.Done()
}
