package gop

import (
	"sync"
	"sync/atomic"
)

type Pool struct {
	conf              Config
	quit              chan struct{}
	tasks             chan TaskFn
	realQueueSize     int32
	additionalWorkers int32
	wg                sync.WaitGroup
}

type TaskFn func()

func NewPool(conf Config) *Pool {
	conf = conf.withDefaults()
	p := &Pool{
		conf:  conf,
		quit:  make(chan struct{}),
		tasks: make(chan TaskFn, conf.MaxQueueSize),
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

	extraWorkers := int32(p.conf.MaxWorkers) - int32(p.conf.UnstoppableWorkers) - atomic.LoadInt32(&p.additionalWorkers)
	currentQueueSize := atomic.LoadInt32(&p.realQueueSize)
	if extraWorkers > 0 && needAdditionalWorker(currentQueueSize, p.conf.MaxQueueSize, p.conf.ExtraWorkersSpawnPercent) {
		w := newAdditionalWorker(p.tasks, p.quit, workerConfig{
			ttl:                   p.conf.ExtraWorkerTTL,
			onTaskTaken:           onTaskTakenWrapper(p.conf.OnTaskTaken, &p.realQueueSize),
			onTaskFinished:        onTaskFinishedWrapper(p.conf.OnTaskFinished, &p.realQueueSize),
			onExtraWorkerSpawned:  onExtraWorkerSpawnedWrapper(p.conf.OnExtraWorkerSpawned, &p.additionalWorkers),
			onExtraWorkerFinished: onExtraWorkerFinishedWrapper(p.conf.OnExtraWorkerFinished, &p.additionalWorkers),
		})
		go w.run()
	}

	select {
	case p.tasks <- t:
		atomic.AddInt32(&p.realQueueSize, 1)
		return nil
	default:
	}

	return ErrPoolFull
}

func (p *Pool) Tasks() <-chan TaskFn {
	return p.tasks
}

func (p *Pool) Run() error {
	select {
	case <-p.quit:
		return ErrPoolClosed
	default:
	}

	workerConf := workerConfig{
		ttl:            p.conf.ExtraWorkerTTL,
		onTaskTaken:    onTaskTakenWrapper(p.conf.OnTaskTaken, &p.realQueueSize),
		onTaskFinished: onTaskFinishedWrapper(p.conf.OnTaskFinished, &p.realQueueSize),
	}
	for i := 0; i < p.conf.UnstoppableWorkers; i++ {
		w := newWorker(p.Tasks(), p.quit, workerConf)
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
