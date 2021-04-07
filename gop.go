// Package gop contains golang worker pool helpers.
package gop

import "context"

// TaskFn is a wrapper for task function.
type TaskFn func()

type Pool interface {
	// Add adds tasks to the pool.
	Add(TaskFn) error
	AddContext(context.Context, TaskFn) error
	// QueueSize is a current queue size.
	QueueSize() int32
	// Shutdown closes pool and stops workers.
	//
	// If any tasks in a queue left, pool will not take them, so the tasks will be lost.
	Shutdown() error
}
