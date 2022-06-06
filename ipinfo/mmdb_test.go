package ipinfo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/internal/qa"
	"github.com/nfx/slrp/pmux"
	"github.com/stretchr/testify/assert"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func TestDonwloadAndLookup(t *testing.T) {
	tempDir := t.TempDir()
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata")))
	defer server.Close()

	// emulate donwloading and unpacking of an updated database tarball
	geoLiteCityFmt = fmt.Sprintf("%s/fake.tar.gz?a=%%s&b=%%s", server.URL)

	l := NewLookup()
	err := l.Configure(app.Config{
		"mmdb_asn":  fmt.Sprintf("%s/asn.mmdb", tempDir),
		"mmdb_city": fmt.Sprintf("%s/city.mmdb", tempDir),
		"license":   "x",
	})
	assert.NoError(t, err)

	// this is a fake IP and fake port, just to do testing.
	// it has nothing to do with the real address.
	info := l.Get(pmux.HttpProxy("1.0.0.100:56789"))
	assert.Equal(t, "ZW / Zimbabwe City / ZimbabweNet", info.String())
}

func TestCreateDummyMmdb(t *testing.T) {
	qa.RunOnlyInDebug(t)

	writer, err := mmdbwriter.New(mmdbwriter.Options{})
	assert.NoError(t, err)

	// this CIDR is used only for testing purposes
	_, testNet, err := net.ParseCIDR("1.0.0.0/16")
	assert.NoError(t, err)
	err = writer.InsertFunc(testNet, inserter.TopLevelMergeWith(mmdbtype.Map{
		"country": mmdbtype.Map{
			"iso_code": mmdbtype.String("ZW"),
		},
		"city": mmdbtype.Map{
			"names": mmdbtype.Map{
				"en": mmdbtype.String("Zimbabwe City"),
			},
		},
		"autonomous_system_organization": mmdbtype.String("ZimbabweNet"),
		"autonomous_system_number":       mmdbtype.Uint16(123),
	}))
	assert.NoError(t, err)

	tgz, err := os.Create("testdata/fake.tar.gz")
	assert.NoError(t, err)
	defer tgz.Close()

	gz := gzip.NewWriter(tgz)
	defer gz.Close()

	tr := tar.NewWriter(gz)
	defer tr.Close()

	readme := []byte("this is fake.")
	err = tr.WriteHeader(&tar.Header{
		Name: "readme.txt",
		Size: int64(len(readme)),
		Mode: 0755,
	})
	assert.NoError(t, err)
	tr.Write(readme)

	buf := bytes.NewBuffer([]byte{})
	_, err = writer.WriteTo(buf)
	assert.NoError(t, err)

	err = tr.WriteHeader(&tar.Header{
		Name: "fake.mmdb",
		Size: int64(buf.Len()),
		Mode: 0755,
	})
	tr.Write(buf.Bytes())
	assert.NoError(t, err)
}
