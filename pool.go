package gop

import (
	"sync"
	"sync/atomic"
	"time"
)

// TaskFn is a wrapper for task function.
type TaskFn func()

// Pool represents worker pool.
type Pool struct {
	conf                        Config
	tasks                       chan TaskFn
	quit                        chan struct{}
	mu                          sync.Mutex
	ticker                      *time.Ticker
	realQueueSize               int32
	unstoppableWorkersAvailable int32
	additionalWorkersAvailable  int32
}

// NewPool creates a new pool with given configuration params.
func NewPool(conf Config) *Pool {
	conf = conf.withDefaults()
	p := &Pool{
		conf:                        conf,
		quit:                        make(chan struct{}),
		tasks:                       make(chan TaskFn, conf.MaxQueueSize),
		unstoppableWorkersAvailable: int32(conf.UnstoppableWorkers),
		additionalWorkersAvailable:  int32(conf.MaxWorkers - conf.UnstoppableWorkers),
	}

	if conf.TaskScheduleTimeout != 0 {
		p.ticker = time.NewTicker(conf.TaskScheduleTimeout)
	} else {
		p.ticker = &time.Ticker{}
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

// Add adds tasks to the pool.
func (p *Pool) Add(t TaskFn) error {
	return p.add(t)
}

func (p *Pool) add(t TaskFn) error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	started := time.Now()

	uAvail := atomic.LoadInt32(&p.unstoppableWorkersAvailable)
	if uAvail == 0 && p.spawnExtraWorker(t) == nil {
		// All the workers are busy and at least one is available.
		return nil
	}

	select {
	case p.tasks <- t:
		atomic.AddInt32(&p.realQueueSize, 1)
		return nil
	default:
	}

	// If we have no additional workers available, just send the task to the
	// task queue.
	err := p.spawnExtraWorker(t)
	if err == nil {
		return nil
	}

	if p.ticker.C == nil {
		return ErrPoolFull
	}

	left := time.Since(started) - p.conf.TaskScheduleTimeout
	if left <= 0 {
		return ErrScheduleTimeout
	}

	select {
	case p.tasks <- t:
		atomic.AddInt32(&p.realQueueSize, 1)
		return nil
	case <-p.quit:
		return ErrPoolClosed
	case <-time.After(left):
		// Wait till task scheduling drops by timeout.
		return ErrScheduleTimeout
	}
}

func (p *Pool) spawnExtraWorker(t TaskFn) error {
	// FIXME: possible optimization. Add check if
	// additional workers are enabled in configuration.
	p.mu.Lock()
	if p.additionalWorkersAvailable == 0 {
		p.mu.Unlock()
		return ErrPoolFull
	}
	p.additionalWorkersAvailable--
	p.mu.Unlock()

	w := newAdditionalWorker(p.tasks, p.quit, &workerConfig{
		ttl: p.conf.ExtraWorkerTTL,
		onTaskTaken: func() {
			atomic.AddInt32(&p.realQueueSize, -1)

			p.mu.Lock()
			p.additionalWorkersAvailable--
			p.mu.Unlock()

			p.conf.OnTaskTaken()
		},
		onTaskFinished: func() {
			p.mu.Lock()
			p.additionalWorkersAvailable++
			p.mu.Unlock()

			p.conf.OnTaskFinished()
		},
		onExtraWorkerSpawned:  p.conf.OnExtraWorkerSpawned,
		onExtraWorkerFinished: p.conf.OnExtraWorkerFinished,
	})
	go w.run(t)

	return nil
}

// QueueSize is a current queue size.
func (p *Pool) QueueSize() int32 {
	res := atomic.LoadInt32(&p.realQueueSize)
	return res
}

// Shutdown closes pool and stops workers.
//
// If any tasks in a queue left, pool will not take them, so the tasks will be lost.
func (p *Pool) Shutdown() error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	close(p.quit)
	p.ticker.Stop()

	return nil
}
