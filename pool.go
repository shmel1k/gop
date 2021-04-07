package gop

import (
	"context"
	"sync/atomic"
	"time"
)

// pool represents worker pool.
type pool struct {
	conf                        Config
	tasks                       chan TaskFn
	quit                        chan struct{}
	ticker                      *time.Ticker
	realQueueSize               int32
	unstoppableWorkersAvailable int32
	additionalWorkersAvailable  int32
}

// NewPool creates a new pool with given configuration params.
func NewPool(conf Config) Pool {
	conf = conf.withDefaults()
	p := &pool{
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

func (p *pool) Add(t TaskFn) error {
	return p.add(context.Background(), t)
}

func (p *pool) AddContext(ctx context.Context, t TaskFn) error {
	return p.add(ctx, t)
}

func (p *pool) add(ctx context.Context, t TaskFn) error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	case <-ctx.Done():
		return ctx.Err()
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
	case <-ctx.Done():
		return ctx.Err()
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

	left := p.conf.TaskScheduleTimeout - time.Since(started)
	if left <= 0 {
		return ErrScheduleTimeout
	}

	select {
	case p.tasks <- t:
		atomic.AddInt32(&p.realQueueSize, 1)
		return nil
	case <-p.quit:
		return ErrPoolClosed
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(left):
		// Wait till task scheduling drops by timeout.
		return ErrScheduleTimeout
	}
}

func (p *pool) spawnExtraWorker(t TaskFn) error {
	// FIXME: possible optimization. Add check if
	// additional workers are enabled in configuration.
	var swapped bool
	for !swapped {
		v := atomic.LoadInt32(&p.additionalWorkersAvailable)
		if v == 0 {
			return ErrPoolFull
		}
		swapped = atomic.CompareAndSwapInt32(&p.additionalWorkersAvailable, v, v-1)
	}

	w := newAdditionalWorker(p.tasks, p.quit, &workerConfig{
		ttl: p.conf.ExtraWorkerTTL,
		onTaskTaken: func() {
			atomic.AddInt32(&p.realQueueSize, -1)

			p.conf.OnTaskTaken()
		},
		onTaskFinished: func() {
			p.conf.OnTaskFinished()
		},
		onExtraWorkerSpawned: func() {
			p.conf.OnExtraWorkerSpawned()
		},
		onExtraWorkerFinished: func() {
			atomic.AddInt32(&p.additionalWorkersAvailable, 1)

			p.conf.OnExtraWorkerFinished()
		},
	})

	w.conf.onExtraWorkerSpawned()
	go w.run(t)

	return nil
}

func (p *pool) QueueSize() int32 {
	res := atomic.LoadInt32(&p.realQueueSize)
	return res
}

func (p *pool) Shutdown() error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	close(p.quit)
	p.ticker.Stop()

	return nil
}
