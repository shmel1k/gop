package gop

import "errors"

// ErrPool are errors occured during pool usage.
var (
	ErrPoolClosed      = errors.New("pool is closed")
	ErrPoolFull        = errors.New("pool is full")
	ErrScheduleTimeout = errors.New("schedule timeout exceeded")
)
