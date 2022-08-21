package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShErr(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"1.2.3.4", "ip"},
		{"1.2.3.4:9823", "addr:port"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			err := ShErr(fmt.Errorf(tt.in))
			assert.EqualError(t, err, tt.out)
		})
	}
}
