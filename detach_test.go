package onecontext

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestDetach(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel1 := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel1()
	ctx, cancel2 := Detach(ctx)

	assert.NoError(t, ctx.Err())

	select {
	case <-ctx.Done():
		t.FailNow()
	default:
	}

	cancel2()
	assert.Error(t, ctx.Err())
}
