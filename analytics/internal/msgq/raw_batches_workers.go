package msgq

import (
	"context"
	"errors"
	"fmt"
)

// RawBatchesWorkers bounds concurrent writes with one channel and prefetch 1 per worker.
type RawBatchesWorkers struct {
	workers []*RawBatchesConsumer
	cancel  context.CancelFunc
}

func NewRawBatchesWorkers(broker *MessageBroker, persister RawBatchPersister, count int) (*RawBatchesWorkers, error) {
	return newRawBatchesWorkers(broker, persister, count, RawPersistenceQueue)
}

func newRawBatchesWorkers(broker *MessageBroker, persister RawBatchPersister, count int, queue string) (*RawBatchesWorkers, error) {
	if count < 1 {
		return nil, fmt.Errorf("raw persistence workers must be positive")
	}
	pool := &RawBatchesWorkers{}
	for i := 0; i < count; i++ {
		worker, err := newRawBatchesConsumer(broker, persister, queue)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("create raw persistence worker %d: %w", i+1, err)
		}
		pool.workers = append(pool.workers, worker)
	}
	return pool, nil
}

func (p *RawBatchesWorkers) Listen(ctx context.Context) error {
	if p.cancel != nil {
		return fmt.Errorf("raw persistence workers already started")
	}
	ctx, p.cancel = context.WithCancel(ctx)
	for i, worker := range p.workers {
		if err := worker.Listen(ctx); err != nil {
			p.Close()
			return fmt.Errorf("start raw persistence worker %d: %w", i+1, err)
		}
	}
	return nil
}

// Cancel all workers before waiting so shutdown is independent of the worker count.
func (p *RawBatchesWorkers) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	var errs []error
	for _, worker := range p.workers {
		errs = append(errs, worker.Close())
	}
	return errors.Join(errs...)
}
