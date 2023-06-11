package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberRangesNormal(t *testing.T) {
	data := []float64{
		24,
		5430,
		5520,
		8754,
		1332,
		3255,
		3245,
		34,
		37,
		5420,
		4420,
	}

	nr := NumberRanges{
		Name:  "Offered",
		Field: "Offered",
		Getter: func(i int) float64 {
			return data[i]
		},
	}

	b := nr.Bucket()
	for i := range data {
		b.Consume(i)
	}

	f := b.Facet(5)
	assert.Equal(t, "Offered", f.Name)
	assert.Equal(t, "24 .. 1332", f.Top[0].Name)
	assert.Equal(t, "5420 .. 5520", f.Top[1].Name)
	assert.Equal(t, "3245 .. 3255", f.Top[2].Name)
}
