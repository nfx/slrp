package serve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignHost(t *testing.T) {
	ca, err := NewCA()
	assert.NoError(t, err)
	mitm, err := ca.Sign("anything")
	assert.NoError(t, err)
	assert.NotNil(t, mitm)
}
