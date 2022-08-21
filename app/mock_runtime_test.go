package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockStartSpin(t *testing.T) {
	a := newServiceA()
	b := newServiceA()
	_, runtime := MockStartSpin(a, b, struct{}{})
	defer runtime.Stop()

	ctx := runtime.Context()
	assert.Nil(t, ctx.Value(1))
}
