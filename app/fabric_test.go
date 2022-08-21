package app

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFabricStartAndLoadFromBackup(t *testing.T) {
	home := t.TempDir()
	data := fmt.Sprintf("%s/.slrp/data", home)
	err := os.MkdirAll(data, 0o700)
	require.NoError(t, err)

	a := newServiceA()

	// emulate some persisted backed up state
	aState, err := os.OpenFile(fmt.Sprintf("%s/a.bak", data),
		os.O_CREATE|os.O_WRONLY, 0o700)
	require.NoError(t, err)

	go func() {
		// create the fixture with no error
		a.flushed <- nil
	}()
	err = gob.NewEncoder(aState).Encode(a)
	require.NoError(t, err)
	err = aState.Sync()
	require.NoError(t, err)
	err = aState.Close()
	require.NoError(t, err)

	testdata, _ := filepath.Abs("testdata")
	defer envm{
		"APP":  "slrp",
		"HOME": home,
		"PWD":  fmt.Sprintf("%s/e", testdata),
	}.restore()()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fabric := &Fabric{
		Factories: Factories{
			"a": func() *serviceA {
				return a
			},
			"spa": MountSpaUI(os.DirFS(fmt.Sprintf("%s/spa", testdata))),
		},
	}
	go fabric.Start(ctx)

	stateA := <-a.state
	assert.Equal(t, []byte{1}, stateA)

	// load without an error
	a.loaded <- nil

	// flush after a heartbeat
	a.flushed <- nil

	equalJson := func(expected map[string]any) func(actual map[string]any) {
		return func(actual map[string]any) {
			assert.Equal(t, expected, actual)
		}
	}

	type permutation struct {
		Verb    string
		Url     string
		Match   func(map[string]any)
		Status  int
		Error   string
		NotJson string
	}
	tests := []permutation{
		{ // SPA handler
			Status: 200,
			Verb:   "GET",
			Url:    "/real/file.json",
			Match: equalJson(map[string]any{
				"test": true,
			}),
		},
		{ // SPA handler
			Status:  200,
			Verb:    "GET",
			Url:     "/for-react",
			NotJson: `from index.html`,
		},
		{ // Fabric
			Verb:   "GET",
			Url:    "/api",
			Status: 200,
			Match: func(services map[string]any) {
				assert.NotNil(t, services["a"], "must have A service")
				assert.NotNil(t, services["monitor"], "must have servers monitor")
				assert.NotNil(t, services["server"], "must have server itself")
			},
		},
		{ // HttpGet
			Status:  200,
			Verb:    "GET",
			Url:     "/api/a",
			NotJson: `1`,
		},
		{ // HttpGetByID
			Status:  200,
			Verb:    "GET",
			Url:     "/api/a/a",
			NotJson: `"a"`,
		},
		{ // HttpGetByID
			Status:  200,
			Verb:    "GET",
			Url:     "/api/a/a?format=text",
			NotJson: `a`,
		},
		{ // HttpDeleteByID
			Status: 400,
			Verb:   "DELETE",
			Url:    "/api/a/error",
			Match: equalJson(map[string]any{
				"Message": "just error: error",
			}),
		},
		{ // HttpDeleteByID
			Status: 404,
			Verb:   "DELETE",
			Url:    "/api/a/not-found",
			Match: equalJson(map[string]any{
				"Message": "no ID found",
			}),
		},
		{ // HttpDeleteByID
			Status: 500,
			Verb:   "DELETE",
			Url:    "/api/a/soft",
			Match: equalJson(map[string]any{
				"Message": "panic with error: soft",
			}),
		},
		{ // HttpDeleteByID
			Status: 500,
			Verb:   "DELETE",
			Url:    "/api/a/hard",
			Match: equalJson(map[string]any{
				"Message": "very wrong error",
			}),
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.Verb, tt.Url), func(t *testing.T) {
			request, _ := http.NewRequest(tt.Verb, fabric.Url()+tt.Url, nil)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				require.EqualError(t, err, tt.Error)
			}
			assert.Equal(t, tt.Status, response.StatusCode)
			raw, err := io.ReadAll(response.Body)
			if err != nil {
				require.EqualError(t, err, tt.Error)
			}
			defer response.Body.Close()
			var freeForm map[string]any
			json.Unmarshal(raw, &freeForm)
			if len(freeForm) == 0 {
				assert.Equal(t, tt.NotJson, string(raw), "NOT JSON")
			} else if tt.Match != nil {
				tt.Match(freeForm)
			} else {
				t.Errorf("Has response, but mo matcher: %v", freeForm)
			}
		})
	}
}
