package counter

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRollingCounterSerDe(t *testing.T) {
	rc := NewRollingCounter(15, time.Second)
	rc.Add(10)
	rc.Add(20)
	rc.Add(30)

	raw, err := rc.MarshalBinary()
	assert.NoError(t, err)

	rc2 := &RollingCounter{}
	err = rc2.UnmarshalBinary(raw)
	assert.NoError(t, err)
	assert.Equal(t, 60, rc.buf[0])
}

func TestRollingCounterState(t *testing.T) {
	rc := NewRollingCounter(15, time.Second)
	rc.buf = []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	rc.pos = 10

	x := rc.Series()
	assert.Equal(t, []int32{11, 12, 13, 14, 15, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, x)
}

func TestRollingCounter(t *testing.T) {
	t.Skip()
	rc := NewRollingCounter(15, time.Second)
	go func() {
		tick := time.NewTicker(333 * time.Millisecond)
		for {
			select {
			case <-tick.C:
				rc.Increment()
			}
		}
	}()
	for {
		tick := time.NewTicker(1 * time.Second)
		select {
		case <-tick.C:
			log.Printf("Sum: %d", rc.Sum())
		}
	}
}
