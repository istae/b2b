package b2b

import "time"

type timer struct {
	t *time.Timer
	c chan struct{}
}

func NewTimer(td time.Duration) {

	c := make(chan struct{})
	t := &timer{
		c: c,
	}
	if td > 0 {
		t.t = time.NewTimer(td)
		go func() {
			<-t.t.C
			close(c)
		}()
	}
}

func (t *timer) Stop() {
	if t.t != nil {
		t.t.Stop()
	}
	NewOnce().Done()
	close(t.c)
}

func (t *timer) Reset(td time.Duration) {
	if t.t != nil {
		t.t.Reset(td)
	}
}
