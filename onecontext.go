package onecontext

import (
	"context"
	"sync"
	"time"
)

type onecontext struct {
	ctx      context.Context
	ctxs     []context.Context
	done     chan struct{}
	err      error
	errMutex sync.Mutex
}

func Merge(ctx context.Context, ctxs ...context.Context) context.Context {
	o := &onecontext{
		done: make(chan struct{}),
		ctx:  ctx,
		ctxs: ctxs,
	}
	go o.run()
	return o
}

func (o *onecontext) Deadline() (time.Time, bool) {
	max := time.Time{}

	deadline, ok := o.ctx.Deadline()
	if ok {
		max = deadline
	}

	for _, ctx := range o.ctxs {
		deadline, ok := ctx.Deadline()
		if ok {
			max = deadline
		}
	}

	return max, !max.IsZero()
}

func (o *onecontext) Done() <-chan struct{} {
	return o.done
}

func (o *onecontext) Err() error {
	o.errMutex.Lock()
	defer o.errMutex.Unlock()
	return o.err
}

func (o *onecontext) Value(key interface{}) interface{} {
	if value := o.ctx.Value(key); value != nil {
		return value
	}

	for _, ctx := range o.ctxs {
		if value := ctx.Value(key); value != nil {
			return value
		}
	}

	return nil
}

func (o *onecontext) run() {
	once := sync.Once{}

	cancelCtx, cancel := context.WithCancel(context.Background())

	o.runContext(o.ctx, cancelCtx, cancel, &once)
	for _, ctx := range o.ctxs {
		o.runContext(ctx, cancelCtx, cancel, &once)
	}
}

func (o *onecontext) runContext(ctx, cancelCtx context.Context, cancel context.CancelFunc, once *sync.Once) {
	go func() {
		select {
		case <-cancelCtx.Done():
			return
		case <-ctx.Done():
			once.Do(func() {
				cancel()
				o.errMutex.Lock()
				o.err = ctx.Err()
				o.errMutex.Unlock()
				o.done <- struct{}{}
			})
		}
	}()
}
