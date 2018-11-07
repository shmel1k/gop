package gop

import "time"

type worker struct {
	quit  <-chan struct{}
	tasks <-chan TaskFn
	conf  workerConfig
}

type workerConfig struct {
	ttl                   time.Duration
	onTaskTaken           func()
	onTaskFinished        func()
	onExtraWorkerSpawned  func()
	onExtraWorkerFinished func()
}

func newWorker(tasks <-chan TaskFn, quit <-chan struct{}, params workerConfig) *worker {
	return &worker{
		quit:  quit,
		tasks: tasks,
		conf:  params,
	}
}

func (w *worker) run() {
	for {
		select {
		case <-w.quit:
			return
		case t := <-w.tasks:
			w.conf.onTaskTaken()
			if t != nil {
				t()
			}
			w.conf.onTaskFinished()
		}
	}
}

type additionalWorker struct {
	*worker
}

func newAdditionalWorker(tasks <-chan TaskFn, quit <-chan struct{}, params workerConfig) *additionalWorker {
	return &additionalWorker{
		worker: &worker{
			quit:  quit,
			tasks: tasks,
			conf:  params,
		},
	}
}

func (w *additionalWorker) run() {
	ticker := time.NewTicker(w.conf.ttl)
	defer ticker.Stop()

	w.conf.onExtraWorkerSpawned()
	defer w.conf.onExtraWorkerFinished()

	for {
		select {
		case <-w.quit:
			return
		case t := <-w.tasks:
			if t != nil {
				w.conf.onTaskTaken()
				t()
				w.conf.onTaskFinished()
			}
		case <-ticker.C:
			return
		}
	}
}
