package ini

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseINI tests the ParseINI function.
func TestParseINI(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedCfg  Config
		expectedErr  bool
		expectedKeys int
	}{
		{
			name: "ValidINIFile",
			input: `[Section1]
Key1=Value1
Key2=Value2

[Section2]
KeyA=ValueA
KeyB=ValueB
`,
			expectedCfg: Config{
				"Section1": {
					"Key1": "Value1",
					"Key2": "Value2",
				},
				"Section2": {
					"KeyA": "ValueA",
					"KeyB": "ValueB",
				},
			},
			expectedErr:  false,
			expectedKeys: 4,
		},
		{
			name:         "EmptyINIFile",
			input:        "",
			expectedCfg:  Config{},
			expectedErr:  false,
			expectedKeys: 0,
		},
		{
			name: "InvalidINIFile",
			input: `[Section1]
Key1=Value1
MissingKey
`,
			expectedErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file, err := createTempFile(test.input)
			assert.NoError(t, err)
			defer os.Remove(file)

			cfg, err := ParseINI(file)

			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.expectedCfg, cfg)
			assert.Equal(t, test.expectedKeys, countKeys(cfg))
		})
	}
}

// Helper function to create a temporary file with the given content.
func createTempFile(content string) (string, error) {
	file, err := os.CreateTemp("", "testfile*.ini")
	if err != nil {
		return "", err
	}
	_, err = file.WriteString(content)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

// Helper function to count the total number of keys in a Config object.
func countKeys(cfg Config) int {
	count := 0
	for _, section := range cfg {
		count += len(section)
	}
	return count
}
