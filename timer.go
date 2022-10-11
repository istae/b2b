package b2b

import (
	"time"
)

type Timer struct {
	C    chan struct{}
	once *Once
	tt   *time.Timer
	dur  time.Duration
}

func NewTimer(dur time.Duration) *Timer {

	t := &Timer{
		once: NewOnce(),
		dur:  dur,
	}

	t.C = t.once.C

	if dur > 0 {
		t.tt = time.NewTimer(dur)
		go func() {
			<-t.tt.C
			t.Close()
		}()
	}

	return t
}

func (t *Timer) Reset() {
	if t.tt != nil {
		if t.tt.Stop() {
			t.tt.Reset(t.dur)
		}
	}
}

func (t *Timer) Close() {
	t.once.Done()
}
