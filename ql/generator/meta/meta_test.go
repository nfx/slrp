package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsing(t *testing.T) {
	ds, err := Parse("../../../probe/reverify.go", "github.com/nfx/slrp/probe", "inReverify")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	m := map[string]string{}
	for _, f := range ds.Type.Fields {
		m[f.Name] = f.AbstractType()
	}
	assert.Equal(t, map[string]string{
		"Proxy":    "string",
		"Attempt":  "number",
		"After":    "number",
		"Country":  "string",
		"Provider": "string",
		"Failure":  "string",
		"ASN":      "number",
	}, m)
}
