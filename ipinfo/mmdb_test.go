package ipinfo

import (
	"testing"

	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"
)

func TestItWorks(t *testing.T) {
	ii := NewLookup()
	i := ii.Get(pmux.HttpProxy("8.8.8.8:53"))
	assert.Equal(t, "", i.Country)
}
