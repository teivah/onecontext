package onecontext

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReset(t *testing.T) {
	ctx := context.WithValue(context.Background(), key(0), "foo")
	ctx = ResetValues(ctx)
	assert.Nil(t, ctx.Value("foo"))
}
