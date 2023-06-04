package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsing(t *testing.T) {
	meta, err := Parse("../../../probe/reverify.go", "github.com/nfx/slrp/probe", "inReverify")
	assert.NoError(t, err)
	assert.NotNil(t, meta)
}
