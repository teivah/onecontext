// Package onecontext provides a mechanism to merge multiple existing contexts.
package onecontext

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCanceled is the returned when the CancelFunc returned by Merge is called.
var ErrCanceled = errors.New("canceled context")

// OneContext is the struct holding the context grouping logic.
type OneContext struct {
	ctx        context.Context
	ctxs       []context.Context
	done       chan struct{}
	err        error
	errMutex   sync.RWMutex
	cancelFunc context.CancelFunc
	cancelCtx  context.Context
}

// Merge allows to merge multiple contexts.
// It returns the merged context and a CancelFunc to cancel it.
func Merge(ctx context.Context, ctxs ...context.Context) (context.Context, context.CancelFunc) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	o := &OneContext{
		done:       make(chan struct{}),
		ctx:        ctx,
		ctxs:       ctxs,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}
	go o.run()
	return o, cancelFunc
}

// Deadline returns the minimum deadline among all the contexts.
func (o *OneContext) Deadline() (time.Time, bool) {
	min := time.Time{}

	if deadline, ok := o.ctx.Deadline(); ok {
		min = deadline
	}

	for _, ctx := range o.ctxs {
		if deadline, ok := ctx.Deadline(); ok {
			if min.IsZero() || deadline.Before(min) {
				min = deadline
			}
		}
	}
	return min, !min.IsZero()
}

// Done returns a channel for cancellation.
func (o *OneContext) Done() <-chan struct{} {
	return o.done
}

// Err returns the first error raised by the contexts, otherwise a nil error.
func (o *OneContext) Err() error {
	o.errMutex.RLock()
	defer o.errMutex.RUnlock()
	return o.err
}

// Value returns the value associated with the key from one of the contexts.
func (o *OneContext) Value(key interface{}) interface{} {
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

func (o *OneContext) run() {
	if len(o.ctxs) == 1 {
		o.runTwoContexts(o.ctx, o.ctxs[0])
		return
	}

	once := sync.Once{}
	o.runMultipleContexts(o.ctx, &once)
	for _, ctx := range o.ctxs {
		o.runMultipleContexts(ctx, &once)
	}
}

func (o *OneContext) runTwoContexts(ctx1, ctx2 context.Context) {
	go func() {
		select {
		case <-o.cancelCtx.Done():
			o.cancel(ErrCanceled)
		case <-ctx1.Done():
			o.cancel(ctx1.Err())
		case <-ctx2.Done():
			o.cancel(ctx2.Err())
		}
	}()
}

func (o *OneContext) runMultipleContexts(ctx context.Context, once *sync.Once) {
	go func() {
		select {
		case <-o.cancelCtx.Done():
			once.Do(func() {
				o.cancel(ErrCanceled)
			})
		case <-ctx.Done():
			once.Do(func() {
				o.cancel(ctx.Err())
			})
		}
	}()
}

func (o *OneContext) cancel(err error) {
	o.errMutex.Lock()
	o.err = err
	o.errMutex.Unlock()
	close(o.done)
	o.cancelFunc()
}
