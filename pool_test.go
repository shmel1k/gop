package gop

import (
	"sync/atomic"
	"testing"
	"time"
)

func shutdownPool(t *testing.T, p *Pool) {
	if err := p.Shutdown(); err != nil {
		t.Errorf("failed to shutdown pool: %s", err)
	}
}

func TestQueueSize(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       42,
		MaxWorkers:         42,
		UnstoppableWorkers: 42,
	})
	defer shutdownPool(t, pool)

	done := make(chan struct{})
	pool.Add(TaskFn(func() {
		close(done)
	}))
	<-done
	queueSize := pool.QueueSize()
	if queueSize != int32(0) {
		t.Errorf("TestQueueSize: got pool size %v, expected %v", queueSize, 0)
	}
}

func TestQueueWorkers(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       42,
		MaxWorkers:         42,
		UnstoppableWorkers: 42,
	})
	defer shutdownPool(t, pool)

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

func TestPoolShutdown(t *testing.T) {
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
	defer shutdownPool(t, pool)

	for i := 0; i < 5; i++ {
		pool.Add(TaskFn(func() {
			time.Sleep(10 * time.Second)
		}))
	}

	if err := pool.Add(TaskFn(func() {})); err != ErrPoolFull {
		t.Fatalf("add task: want err %v, got %v", ErrPoolFull, err)
	}
}

func TestPoolScheduleTimeout(t *testing.T) {
	var testData = []struct {
		pool        *Pool
		expectedErr error
	}{
		{
			pool: NewPool(Config{
				MaxQueueSize:        1,
				MaxWorkers:          1,
				UnstoppableWorkers:  1,
				TaskScheduleTimeout: 10 * time.Millisecond,
			}),
			expectedErr: ErrScheduleTimeout,
		},
		{
			pool: NewPool(Config{
				MaxQueueSize:        1,
				MaxWorkers:          1,
				UnstoppableWorkers:  1,
				TaskScheduleTimeout: 5 * time.Nanosecond,
			}),
			expectedErr: ErrScheduleTimeout,
		},
	}

	for i, v := range testData {
		for i := 0; i < 5; i++ {
			v.pool.Add(TaskFn(func() {
				time.Sleep(10 * time.Second)
			}))
		}

		if err := v.pool.Add(TaskFn(func() {})); err != v.expectedErr {
			t.Errorf("TestPoolScheduleTimeout[%d]: add task: want err %v, got %v", i, v.expectedErr, err)
		}
		shutdownPool(t, v.pool)
	}
}

func TestPoolWithAdditionalWorkers(t *testing.T) {
	var started int32
	var finished int32

	done := make(chan struct{})

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
			done <- struct{}{}
		},
	})
	defer shutdownPool(t, pool)

	var err error
	for i := 0; i < 3; i++ {
		err = pool.Add(TaskFn(func() {
			time.Sleep(300 * time.Millisecond)
		}))
		if err != nil {
			t.Fatal(err)
		}
	}

	<-done
	<-done
}

func TestPoolWithOnlyAdditionalWorkers(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         1,
		UnstoppableWorkers: 0,
		ExtraWorkerTTL:     300 * time.Millisecond,
	})
	defer shutdownPool(t, pool)

	done := make(chan struct{})
	err := pool.Add(TaskFn(func() {
		done <- struct{}{}
	}))
	if err != nil {
		t.Fatal(err)
	}
	err = pool.Add(TaskFn(func() {
		done <- struct{}{}
	}))
	if err != nil {
		t.Fatal(err)
	}
	<-done
	<-done

}

func TestPoolCloseAdditionalWorker(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         1,
		UnstoppableWorkers: 0,
		ExtraWorkerTTL:     100 * time.Millisecond,
	})
	defer shutdownPool(t, pool)

	done := make(chan struct{})
	err := pool.Add(TaskFn(func() {
		close(done)
	}))
	if err != nil {
		t.Fatal(err)
	}

	<-done

}

func TestPoolCloseAfterWorkerTask(t *testing.T) {
	pool := NewPool(Config{
		MaxQueueSize:       1,
		MaxWorkers:         1,
		UnstoppableWorkers: 0,
		ExtraWorkerTTL:     time.Minute,
	})
	done := make(chan struct{})
	err := pool.Add(TaskFn(func() {
		select {
		case <-time.After(10 * time.Millisecond):
			close(done)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}

	defer shutdownPool(t, pool)
	<-done
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
	defer shutdownPool(t, pool)

	pool.Add(TaskFn(func() {
		time.Sleep(150 * time.Millisecond)
	}))
	pool.Add(TaskFn(func() {
	}))
}

func BenchmarkPool(b *testing.B) {
	pool := NewPool(Config{
		MaxQueueSize:       0,
		MaxWorkers:         20,
		UnstoppableWorkers: 20,
	})
	defer pool.Shutdown()

	for i := 0; i < b.N; i++ {
		pool.Add(TaskFn(func() {
		}))
	}
}
