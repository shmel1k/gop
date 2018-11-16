package gop

import (
	"sync/atomic"
)

type TaskFn func()

type Pool struct {
	conf                        Config
	tasks                       chan TaskFn
	quit                        chan struct{}
	realQueueSize               int32
	unstoppableWorkersAvailable int32
	additionalWorkersAvailable  int32
}

func NewPool(conf Config) *Pool {
	conf = conf.withDefaults()
	p := &Pool{
		conf:                        conf,
		quit:                        make(chan struct{}),
		tasks:                       make(chan TaskFn, conf.MaxQueueSize),
		unstoppableWorkersAvailable: int32(conf.UnstoppableWorkers),
		additionalWorkersAvailable:  int32(conf.MaxWorkers - conf.UnstoppableWorkers),
	}

	workerConf := workerConfig{
		onTaskTaken: func() {
			atomic.AddInt32(&p.realQueueSize, -1)
			atomic.AddInt32(&p.unstoppableWorkersAvailable, -1)
			p.conf.OnTaskTaken()
		},
		onTaskFinished: func() {
			atomic.AddInt32(&p.unstoppableWorkersAvailable, 1)
			p.conf.OnTaskFinished()
		},
	}

	for i := 0; i < p.conf.UnstoppableWorkers; i++ {
		w := newWorker(p.tasks, p.quit, &workerConf)
		go w.run()
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

	uAvail := atomic.LoadInt32(&p.unstoppableWorkersAvailable)
	aAvail := atomic.LoadInt32(&p.additionalWorkersAvailable)
	if uAvail == 0 && aAvail > 0 {
		// All the workers are busy and at least one is available.
		return p.spawnExtraWorker(t)
	}

	select {
	case p.tasks <- t:
		atomic.AddInt32(&p.realQueueSize, 1)
		return nil
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	aAvail = atomic.LoadInt32(&p.additionalWorkersAvailable)
	if aAvail > 0 {
		return p.spawnExtraWorker(t)
	}

	return ErrPoolFull
}

func (p *Pool) spawnExtraWorker(t TaskFn) error {
	atomic.AddInt32(&p.additionalWorkersAvailable, -1)
	w := newAdditionalWorker(p.tasks, p.quit, &workerConfig{
		ttl: p.conf.ExtraWorkerTTL,
		onTaskTaken: func() {
			atomic.AddInt32(&p.realQueueSize, -1)
			atomic.AddInt32(&p.additionalWorkersAvailable, -1)
			p.conf.OnTaskTaken()
		},
		onTaskFinished:       p.conf.OnTaskFinished,
		onExtraWorkerSpawned: p.conf.OnExtraWorkerSpawned,
		onExtraWorkerFinished: func() {
			atomic.AddInt32(&p.additionalWorkersAvailable, 1)
			p.conf.OnExtraWorkerFinished()
		},
	})
	go w.run(t)

	return nil
}

func (p *Pool) QueueSize() int32 {
	res := atomic.LoadInt32(&p.realQueueSize)
	return res
}

func (p *Pool) Shutdown() error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	close(p.quit)

	return nil
}
