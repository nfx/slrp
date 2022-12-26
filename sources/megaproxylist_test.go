package sources

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMegaproxylist(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata/megaproxylist")))
	defer server.Close()
	megaproxylistUrl = fmt.Sprintf("%s/test.zip", server.URL)
	testSource(t, func(ctx context.Context) Src {
		return ByID(65).Feed(ctx, http.DefaultClient)
	}, 3)
}

func Test_unzipInMemory(t *testing.T) {
	rFile, _ := os.ReadFile("./testdata/megaproxylist/test.zip")
	data, _ := unzipInMemory(rFile)
	assert.Equal(t, 100, len(data))
}

func Test_readZipFile(t *testing.T) {
	rFile, _ := os.ReadFile("./testdata/megaproxylist/test.zip")
	zipReader, _ := zip.NewReader(bytes.NewReader(rFile), int64(len(rFile)))
	fmt.Println(zipReader)
	assert.Equal(t, zipReader.File[0].Name, "megaproxylist.csv", "Testing unzip in memory")
}
