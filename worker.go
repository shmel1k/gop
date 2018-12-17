package gop

import (
	"time"
)

type worker struct {
	quit  <-chan struct{}
	tasks <-chan TaskFn
	conf  *workerConfig
}

type workerConfig struct {
	ttl                   time.Duration
	onTaskTaken           func()
	onTaskFinished        func()
	onExtraWorkerSpawned  func()
	onExtraWorkerFinished func()
}

func newWorker(tasks <-chan TaskFn, quit <-chan struct{}, params *workerConfig) *worker {
	return &worker{
		quit:  quit,
		tasks: tasks,
		conf:  params,
	}
}

func (w *worker) run() {
	for {
		select {
		case t := <-w.tasks:
			w.runTask(t)
		case <-w.quit:
			return
		}
	}
}

func (w *worker) runTask(t TaskFn) {
	if t != nil {
		w.conf.onTaskTaken()
		t()
		w.conf.onTaskFinished()
	}
}

type additionalWorker worker

func newAdditionalWorker(tasks <-chan TaskFn, quit <-chan struct{}, params *workerConfig) *additionalWorker {
	return &additionalWorker{
		quit:  quit,
		tasks: tasks,
		conf:  params,
	}
}

func (w *additionalWorker) run(t TaskFn) {
	// Execute the task anyway.
	// But next we can easily stop the worker.

	ticker := time.NewTicker(w.conf.ttl)
	defer ticker.Stop()

	w.conf.onExtraWorkerSpawned()
	defer w.conf.onExtraWorkerFinished()

	w.runTask(t)

	for {
		select {
		case <-ticker.C:
			return
		case <-w.quit:
			return
		case t := <-w.tasks:
			w.runTask(t)
		}
	}
}

func (w *additionalWorker) runTask(t TaskFn) {
	if t != nil {
		w.conf.onTaskTaken()
		t()
		w.conf.onTaskFinished()
	}
}
