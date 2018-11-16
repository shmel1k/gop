package gop

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueSize(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       42,
		MaxWorkers:         2,
		UnstoppableWorkers: 2,
	})

	err := pool.Add(TaskFn(func() {
		time.Sleep(1 * time.Second)
	}))
	if err != nil {
		t.Fatal(err)
	}

	err = pool.Add(TaskFn(func() {
		time.Sleep(1 * time.Second)
	}))
	if err != nil {
		t.Fatal(err)
	}

	size := pool.QueueSize()
	if size != 2 {
		t.Fatalf("invalid queue size: got %v, want %v", size, 2)
	}
	pool.Shutdown()
}

func TestQueueWorkers(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       42,
		MaxWorkers:         42,
		UnstoppableWorkers: 42,
	})

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
	pool := NewPool(Config{})

	time.Sleep(time.Millisecond)
	err := pool.Add(TaskFn(func() {
	}))
	if err != nil {
		t.Fatal(err)
	}

	err = pool.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	err = pool.Add(TaskFn(func() {
	}))
	if err != ErrPoolClosed {
		t.Fatalf("add task: want err %v, got %v", ErrPoolClosed, err)
	}
	if err = pool.Shutdown(); err != ErrPoolClosed {
		t.Fatalf("close pool: want err %v, got %v", ErrPoolClosed, err)
	}
}

func TestPoolQueueOverfilled(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         1,
		UnstoppableWorkers: 1,
	})

	for i := 0; i < 5; i++ {
		pool.Add(TaskFn(func() {
			time.Sleep(10 * time.Second)
		}))
	}

	if err := pool.Add(TaskFn(func() {})); err != ErrPoolFull {
		t.Fatalf("add task: want err %v, got %v", ErrPoolFull, err)
	}
}

func TestPoolWithAdditionalWorkers(t *testing.T) {
	var started int32
	var finished int32

	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         3,
		UnstoppableWorkers: 1,
		ExtraWorkerTTL:     300 * time.Millisecond,
		OnExtraWorkerSpawned: func() {
			atomic.AddInt32(&started, 1)
		},
		OnExtraWorkerFinished: func() {
			atomic.AddInt32(&finished, 1)
		},
	})

	var err error
	for i := 0; i < 3; i++ {
		err = pool.Add(TaskFn(func() {
			time.Sleep(300 * time.Millisecond)
		}))
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(500 * time.Millisecond)

	st := atomic.LoadInt32(&started)
	if st != 2 {
		t.Fatalf("additional worker started: want %v, got %v", 2, st)
	}

	fin := atomic.LoadInt32(&finished)
	if fin != 2 {
		t.Fatalf("additional worker finished: want %v, got %v", 2, fin)
	}
}

func TestPoolWithOnlyAdditionalWorkers(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         1,
		UnstoppableWorkers: 0,
		ExtraWorkerTTL:     300 * time.Millisecond,
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
	time.Sleep(100 * time.Millisecond)
	pool.Shutdown()
}

func TestPoolCloseAdditionalWorker(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         1,
		UnstoppableWorkers: 0,
		ExtraWorkerTTL:     100 * time.Millisecond,
	})
	err := pool.Add(TaskFn(func() {
	}))
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)
	pool.Shutdown()
}

func TestPoolCloseAfterWorkerTask(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         1,
		UnstoppableWorkers: 0,
		ExtraWorkerTTL:     time.Minute,
	})
	err := pool.Add(TaskFn(func() {
		time.Sleep(20 * time.Millisecond)
	}))
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)

	pool.Shutdown()
	time.Sleep(500 * time.Millisecond)
}

func TestPoolWithAdditionalWorkersClose(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:          1,
		MaxWorkers:            2,
		UnstoppableWorkers:    1,
		ExtraWorkerTTL:        500 * time.Millisecond,
		OnExtraWorkerSpawned:  func() {},
		OnExtraWorkerFinished: func() {},
	})

	pool.Add(TaskFn(func() {
		time.Sleep(150 * time.Millisecond)
	}))
	pool.Add(TaskFn(func() {
	}))
	pool.Shutdown()
}

func BenchmarkPool(b *testing.B) {
	pool := NewPool(Config{
		MaxQueueSize:       0,
		MaxWorkers:         10,
		UnstoppableWorkers: 10,
	})

	defer pool.Shutdown()
	for i := 0; i < b.N; i++ {
		pool.Add(TaskFn(func() {
		}))
	}
}
