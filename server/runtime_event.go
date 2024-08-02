package server

import (
	"context"

	"go.uber.org/zap"
)

type RuntimeEventQueue struct {
	logger  *zap.Logger
	metrics Metrics

	ch chan func()

	ctx         context.Context
	ctxCancelFn context.CancelFunc
}

func NewRuntimeEventQueue(logger *zap.Logger, config Config, metrics Metrics) *RuntimeEventQueue {
	b := &RuntimeEventQueue{
		logger:  logger,
		metrics: metrics,

		ch: make(chan func(), config.GetRuntime().EventQueueSize),
	}
	b.ctx, b.ctxCancelFn = context.WithCancel(context.Background())

	// Start a fixed number of workers.
	for i := 0; i < config.GetRuntime().EventQueueWorkers; i++ {
		go func() {
			for {
				select {
				case <-b.ctx.Done():
					return
				case fn := <-b.ch:
					fn()
				}
			}
		}()
	}

	return b
}

func (b *RuntimeEventQueue) Queue(fn func()) {
	select {
	case b.ch <- fn:
		// Event queued successfully.
	default:
		// Event queue is full, drop it to avoid blocking the caller.
		b.metrics.CountDroppedEvents(1)
		b.logger.Warn("Runtime event queue full, events may be lost")
	}
}

func (b *RuntimeEventQueue) Stop() {
	b.ctxCancelFn()
}
