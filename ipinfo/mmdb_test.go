package ipinfo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/internal/qa"
	"github.com/nfx/slrp/pmux"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"

	"github.com/stretchr/testify/assert"
)

type envm map[string]string

func (env envm) apply() {
	for k, v := range env {
		os.Setenv(k, v)
	}
}

func (env envm) restore() func() {
	backup := envm{}
	for _, line := range os.Environ() {
		pair := strings.SplitN(line, "=", 2)
		backup[pair[0]] = pair[1]
	}
	os.Clearenv()
	env.apply()
	return func() {
		backup.apply()
	}
}

func TestNoFileNoLicense(t *testing.T) {
	defer envm{
		"HOME": t.TempDir(),
	}.restore()()

	l := &Lookup{}
	l.Start(nil)
	err := l.Configure(app.Config{})
	assert.NoError(t, err)
}

func TestWrongFile(t *testing.T) {
	dir := t.TempDir()
	defer envm{
		"APP":  "slrp",
		"HOME": dir,
	}.restore()()

	// has to be file, but we give it a dir
	path := filepath.Join(dir, ".slrp", "maxmind")
	err := os.MkdirAll(path, 0o700)
	assert.NoError(t, err)

	f, err := os.OpenFile(filepath.Join(path, "GeoLite2-City.mmdb"),
		os.O_CREATE|os.O_WRONLY, 0o700)
	assert.NoError(t, err)
	f.WriteString("nope")
	f.Close()

	l := &Lookup{}
	l.Start(nil)
	err = l.Configure(app.Config{})
	assert.EqualError(t, err, "error opening database: invalid MaxMind DB file")
}

func TestNotFoundDownload(t *testing.T) {
	tempDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
	defer server.Close()

	// emulate donwloading and unpacking of an updated database tarball
	geoLiteCityFmt = fmt.Sprintf("%s/none?a=%%s&b=%%s", server.URL)

	l := NewLookup()
	err := l.Configure(app.Config{
		"mmdb_asn":  fmt.Sprintf("%s/asn.mmdb", tempDir),
		"mmdb_city": fmt.Sprintf("%s/city.mmdb", tempDir),
		"license":   "x",
	})
	assert.EqualError(t, err, "cannot download GeoLite2-City: EOF")
}

func TestNotFoundDownloadBadUrl(t *testing.T) {
	tempDir := t.TempDir()

	// emulate donwloading and unpacking of an updated database tarball
	geoLiteCityFmt = fmt.Sprintf("%s/%s?a=%%s&b=%%s",
		fmt.Sprint(0x7f),
		"http://127.0.0.1")

	l := NewLookup()
	err := l.Configure(app.Config{
		"mmdb_asn":  fmt.Sprintf("%s/asn.mmdb", tempDir),
		"mmdb_city": fmt.Sprintf("%s/city.mmdb", tempDir),
		"license":   "x",
	})
	assert.Error(t, err)
}

func TestDonwloadBadTarGz(t *testing.T) {
	tempDir := t.TempDir()
	server := httptest.NewServer(http.FileServer(http.Dir("./testdata")))
	defer server.Close()

	// emulate donwloading and unpacking of an updated database tarball
	geoLiteCityFmt = fmt.Sprintf("%s/empty.gz?a=%%s&b=%%s", server.URL)

	l := NewLookup()
	err := l.Configure(app.Config{
		"mmdb_asn":  fmt.Sprintf("%s/asn.mmdb", tempDir),
		"mmdb_city": fmt.Sprintf("%s/city.mmdb", tempDir),
		"license":   "x",
	})
	assert.EqualError(t, err, "cannot download GeoLite2-City: cannot find .mmdb file in archive")
}

func copyFile(src, dst string) error {
	raw, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, raw, 0o600)
}

func TestFilesWereThereAlready(t *testing.T) {
	tempDir := t.TempDir()
	asnFile := fmt.Sprintf("%s/asn.mmdb", tempDir)
	cityFile := fmt.Sprintf("%s/city.mmdb", tempDir)

	copyFile("testdata/fake.mmdb", asnFile)
	copyFile("testdata/fake.mmdb", cityFile)

	l := NewLookup()
	err := l.Configure(app.Config{
		"mmdb_asn":  asnFile,
		"mmdb_city": cityFile,
	})
	assert.NoError(t, err)

	// this is a fake IP and fake port, just to do testing.
	// it has nothing to do with the real address.
	info := l.Get(pmux.HttpProxy("1.0.0.100:56789"))
	assert.Equal(t, "ZW / Zimbabwe City / ZimbabweNet", info.String())
}

func TestDonwloadValidAndCannotMakeDir(t *testing.T) {
	tempDir := "/dev/null"
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
	assert.EqualError(t, err, "cannot mkdir for GeoLite2-City: mkdir /dev/null: not a directory")
}

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
