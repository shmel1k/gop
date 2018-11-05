package gop

import "errors"

var (
	ErrPoolClosed = errors.New("pool is closed")
	ErrPoolFull   = errors.New("pool is full")
)
