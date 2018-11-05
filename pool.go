package gop

import (
	"sync"
)

type Pool struct {
	conf      Config
	quit      chan struct{}
	tasksChan chan TaskFn
	wg        sync.WaitGroup
}

type TaskFn func()

func NewPool(conf Config) (*Pool, error) {
	conf = conf.withDefaults()
	p := &Pool{
		conf: conf,
		quit: make(chan struct{}),
	}

	if conf.MaxQueueSize == 0 {
		p.tasks = make(chan TaskFn)
	} else {
		p.tasks = make(chan TaskFn, conf.MaxQueueSize)
	}

	return p, nil
}

func (p *Pool) Add(t TaskFn) error {
	return p.add(t)
}

func (p *Pool) add(t TaskFn) error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	case p.tasks <- t:
		return nil
	default:
	}

	return ErrPoolFull
}

func (p *Pool) tasks() <-chan TaskFn {
	return p.tasksChan
}

func (p *Pool) Run() {
	for i := 0; i < p.conf.MaxWorkers; i++ {
		go p.worker()
	}
}

func (p *Pool) Close() error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	close(p.quit)
	p.wg.Wait()
	close(p.tasks)

	return nil
}
