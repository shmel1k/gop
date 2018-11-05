package gop

import (
	"sync"
	"time"
)

type worker struct {
	quit          chan struct{}
	done          chan struct{}
	realQueueSize int32
	wg            sync.WaitGroup
	tasks         <-chan TaskFn
}

func newWorker(subscription <-chan TaskFn) *worker {
	return &worker{
		quit:  make(chan struct{}),
		tasks: subscription,
	}
}

func (w *worker) run() {
	for {
		select {
		case <-w.quit:
			close(w.done)
			return
		case t := <-w.tasks:
			t()
		}
	}
}

func (w *worker) stop() {
	close(w.quit)
	<-w.done
}

type additionalWorker struct {
	*worker
	ttl time.Duration
}

func newAdditionalWorker(subscription <-chan TaskFn, ttl time.Duration) *additionalWorker {
	return &additionalWorker{
		worker: &worker{
			done:  make(chan struct{}),
			quit:  make(chan struct{}),
			tasks: subscription,
		},
		ttl: ttl,
	}
}

func (w *additionalWorker) run() error {
	for {
		select {
		case <-w.quit:
			close(w.done)
			return nil
		case t := <-w.tasks:
			t()
		}
	}
	return nil
}
