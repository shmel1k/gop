package gop

var (
	defaultMaxWorkers         = 10
	defaultUnstoppableWorkers = 10
)

type Config struct {
	MaxWorkers         int
	UnstoppableWorkers int

	// MaxQueueSize defines maximum work
	// queue size. If MaxQueueSize is 0, then queue is unlimited.
	MaxQueueSize int
}

func (c Config) withDefaults() Config {
	if c.MaxWorkers == 0 {
		c.MaxWorkers = defaultMaxWorkers
	}
	if c.UnstoppableWorkers == 0 {
		c.UnstoppableWorkers = defaultMaxWorkers
	}
	return c
}
