package gop

import "time"

var (
	defaultMaxWorkers         = 10
	defaultUnstoppableWorkers = 10
	defaultWorkerTTL          = time.Second
)

// Config is a configuration parameters for pool.
type Config struct {
	// MaxWorkers is a number of workers run until the pool is stopped.
	// In this count also included extra workers that added if the pool
	// queue runs out of limit.
	MaxWorkers int

	// UnstoppableWorkers is a number of workers that run
	// forever until the pool is stopped.
	UnstoppableWorkers int

	// MaxQueueSize defines maximum work
	// queue size. If MaxQueueSize is 0, then queue is unlimited.
	MaxQueueSize int

	// ExtraWorkerTTL determines the timeout after which
	// extra worker shuts down.
	ExtraWorkerTTL time.Duration

	// TaskScheduleTimeout determines the timeout
	// for task to be added to the queue.
	TaskScheduleTimeout time.Duration

	// OnTaskTaken determines a callback is called after any task
	// from queue is taken.
	OnTaskTaken func()

	// OnTaskFinished determines a callback is called after any task
	// is completed.
	OnTaskFinished func()

	// OnExtraWorkerSpawned determines a callback is called after
	// extra worker is spawned.
	OnExtraWorkerSpawned func()

	// OnExtraWorkerFinished determines a callback is called
	// after extra worker is finished.
	OnExtraWorkerFinished func()
}

func (c Config) withDefaults() Config {
	if c.MaxWorkers == 0 {
		c.MaxWorkers = defaultMaxWorkers
	}
	if c.ExtraWorkerTTL == 0 {
		c.ExtraWorkerTTL = defaultWorkerTTL
	}
	if c.OnTaskTaken == nil {
		c.OnTaskTaken = func() {}
	}
	if c.OnTaskFinished == nil {
		c.OnTaskFinished = func() {}
	}
	if c.OnExtraWorkerSpawned == nil {
		c.OnExtraWorkerSpawned = func() {}
	}
	if c.OnExtraWorkerFinished == nil {
		c.OnExtraWorkerFinished = func() {}
	}
	return c
}
