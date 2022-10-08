package b2b

import (
	"sync"
)

type Once struct {
	C    chan struct{}
	done uint32
	mtx  sync.Mutex
}

func NewOnce() *Once {
	return &Once{C: make(chan struct{})}
}

func (o *Once) Done() {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	if o.done == 0 {
		close(o.C)
	}
	o.done = 1
}
