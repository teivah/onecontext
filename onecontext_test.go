package onecontext

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

type key int

const (
	foo key = iota
	bar
	baz
)

func eventually(ch <-chan struct{}) bool {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelFunc()

	select {
	case <-ch:
		return true
	case <-timeout.Done():
		return false
	}
}

func Test_Merge_Nominal(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx1, cancel1 := context.WithCancel(context.WithValue(context.Background(), foo, "foo"))
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.WithValue(context.Background(), bar, "bar"))

	ctx, _ := Merge(ctx1, ctx2)

	deadline, ok := ctx.Deadline()
	assert.True(t, deadline.IsZero())
	assert.False(t, ok)

	assert.Equal(t, "foo", ctx.Value(foo))
	assert.Equal(t, "bar", ctx.Value(bar))
	assert.Nil(t, ctx.Value(baz))

	assert.False(t, eventually(ctx.Done()))
	assert.NoError(t, ctx.Err())

	cancel2()
	assert.True(t, eventually(ctx.Done()))
	assert.Error(t, ctx.Err())
}

func Test_Merge_Deadline_Context1(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	ctx2 := context.Background()

	ctx, _ := Merge(ctx1, ctx2)

	deadline, ok := ctx.Deadline()
	assert.False(t, deadline.IsZero())
	assert.True(t, ok)
}

func Test_Merge_Deadline_Context2(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx1 := context.Background()
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()

	ctx, _ := Merge(ctx1, ctx2)

	deadline, ok := ctx.Deadline()
	assert.False(t, deadline.IsZero())
	assert.True(t, ok)
}

func Test_Merge_Deadline_ContextN(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx1 := context.Background()
	ctxs := make([]context.Context, 0)
	for i := 0; i < 10; i++ {
		ctx := context.Background()
		ctxs = append(ctxs, ctx)
	}
	ctxN, cancel := context.WithTimeout(context.Background(), time.Second)
	ctxs = append(ctxs, ctxN)
	for i := 0; i < 10; i++ {
		ctx := context.Background()
		ctxs = append(ctxs, ctx)
	}

	ctx, _ := Merge(ctx1, ctxs...)

	assert.False(t, eventually(ctx.Done()))
	assert.NoError(t, ctx.Err())

	cancel()
	assert.True(t, eventually(ctx.Done()))
	assert.Error(t, ctx.Err())
}

func Test_Merge_Deadline_None(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx1 := context.Background()
	ctx2 := context.Background()

	ctx, cancel := Merge(ctx1, ctx2)
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, deadline.IsZero())
	assert.False(t, ok)
}

func Test_Cancel_Two(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx1 := context.Background()
	ctx2 := context.Background()

	ctx, cancel := Merge(ctx1, ctx2)

	cancel()
	assert.True(t, eventually(ctx.Done()))
	assert.Error(t, ctx.Err())
	assert.Equal(t, "canceled context", ctx.Err().Error())
	assert.Equal(t, ErrCanceled, ctx.Err())
}

func Test_Cancel_Multiple(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx1 := context.Background()
	ctx2 := context.Background()
	ctx3 := context.Background()

	ctx, cancel := Merge(ctx1, ctx2, ctx3)

	cancel()
	assert.True(t, eventually(ctx.Done()))
	assert.Error(t, ctx.Err())
	assert.Equal(t, "canceled context", ctx.Err().Error())
	assert.Equal(t, ErrCanceled, ctx.Err())
}
