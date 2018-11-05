package gop

type worker struct {
	quit  <-chan struct{}
	tasks <-chan TaskFn
}

func newWorker(subscription <-chan TaskFn, quit <-chan struct{}) *worker {
	return &worker{
		quit:  quit,
		tasks: subscription,
	}
}

func (w *worker) run() {
	for {
		select {
		case <-w.quit:
			return
		case t := <-w.tasks:
			if t != nil {
				t()
			}
		}
	}
}
