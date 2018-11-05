package gop

import (
	"sync/atomic"
	"testing"
)

func TestQueueSize(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       42,
		MaxWorkers:         42,
		UnstoppableWorkers: 42,
	})
	err := pool.Add(TaskFn(func() {
	}))
	if err != nil {
		t.Fatal(err)
	}

	err = pool.Add(TaskFn(func() {
	}))
	if err != nil {
		t.Fatal(err)
	}

	size := pool.QueueSize()
	if size != 2 {
		t.Fatalf("invalid queue size: got %v, want %v", size, 2)
	}
	pool.Close()
}

func TestQueueWorkers(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       42,
		MaxWorkers:         42,
		UnstoppableWorkers: 42,
	})
	pool.Run()

	done := make(chan struct{}, 2)
	var res int32
	pool.Add(TaskFn(func() {
		atomic.AddInt32(&res, 1)
		done <- struct{}{}
	}))
	pool.Add(TaskFn(func() {
		atomic.AddInt32(&res, 1)
		done <- struct{}{}
	}))
	<-done
	<-done
	resFinal := atomic.LoadInt32(&res)

	if res != int32(2) {
		t.Errorf("invalid result size: got %v, want %v", resFinal, 2)
	}
}

func TestPoolClose(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       0,
		MaxWorkers:         42,
		UnstoppableWorkers: 42,
	})
	pool.Run()
	err := pool.Add(TaskFn(func() {
	}))
	if err != nil {
		t.Fatal(err)
	}

	err = pool.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = pool.Add(TaskFn(func() {
	}))
	if err != ErrPoolClosed {
		t.Fatalf("add task: want err %v, got %v", ErrPoolClosed, err)
	}
	if err = pool.Close(); err != ErrPoolClosed {
		t.Fatalf("close pool: want err %v, got %v", ErrPoolClosed, err)
	}
	if err = pool.Run(); err != ErrPoolClosed {
		t.Fatalf("run pool: want err %v, got %v", ErrPoolClosed, err)
	}
}

func TestPoolQueueOverfilled(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         0,
		UnstoppableWorkers: 0,
	})
	defer pool.Close()
	pool.Add(TaskFn(func() {
	}))

	if err := pool.Add(TaskFn(func() {})); err != ErrPoolFull {
		t.Fatalf("add task: want err %v, got %v", err, ErrPoolFull)
	}
}

func BenchmarkPool(b *testing.B) {
	pool := NewPool(Config{
		MaxQueueSize:       0,
		MaxWorkers:         10,
		UnstoppableWorkers: 10,
	})
	pool.Run()
	defer pool.Close()
	for i := 0; i < b.N; i++ {
		pool.Add(TaskFn(func() {
		}))
	}
}
