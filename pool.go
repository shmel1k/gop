package gop

import (
	"sync"
	"sync/atomic"
)

type Pool struct {
	conf          Config
	quit          chan struct{}
	tasksChan     chan TaskFn
	realQueueSize int32
	wg            sync.WaitGroup
}

type TaskFn func()

func NewPool(conf Config) *Pool {
	conf = conf.withDefaults()
	p := &Pool{
		conf:      conf,
		quit:      make(chan struct{}),
		tasksChan: make(chan TaskFn, conf.MaxQueueSize),
	}

	return p
}

func (p *Pool) Add(t TaskFn) error {
	return p.add(t)
}

func (p *Pool) add(t TaskFn) error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	select {
	case p.tasksChan <- t:
		atomic.AddInt32(&p.realQueueSize, 1)
		return nil
	default:
	}

	return ErrPoolFull
}

func (p *Pool) tasks() <-chan TaskFn {
	return p.tasksChan
}

func (p *Pool) Run() error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	for i := 0; i < p.conf.UnstoppableWorkers; i++ {
		w := newWorker(p.tasks(), p.quit)
		go w.run()
	}

	return nil
}

func (p *Pool) QueueSize() int32 {
	res := atomic.LoadInt32(&p.realQueueSize)
	return res
}

func (p *Pool) Close() error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	close(p.quit)
	p.wg.Wait()

	return nil
}
