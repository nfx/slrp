package ipinfo

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/pmux"
	mmdb "github.com/oschwald/maxminddb-golang"
)

var _ app.Service = &Lookup{}

type Lookup struct {
	city *mmdb.Reader
	asn  *mmdb.Reader
}

func NewLookup() *Lookup {
	return &Lookup{}
}

func (i *Lookup) Configure(c app.Config) error {
	city, err := i.ensureDownloaded(c, "GeoLite2-City")
	if err == errNoLicence {
		// no license - no downloads, but files might be added normally
		return nil
	}
	if err != nil {
		return err
	}
	i.city = city
	asn, err := i.ensureDownloaded(c, "GeoLite2-ASN")
	if err != nil {
		return err
	}
	i.asn = asn
	return nil
}

func (i *Lookup) Start(ctx app.Context) {
	// noop - later, when we'll be doing refreshes - we should decide if we
	// should block or not, replace reader just from one thread and etc.
}

var geoLiteCityFmt = "https://download.maxmind.com/app/geoip_download?edition_id=%s&license_key=%s&suffix=tar.gz"

func (i *Lookup) download(edition, licence string) ([]byte, error) {
	url := fmt.Sprintf(geoLiteCityFmt, edition, licence) // GeoLite2-City
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	inner := tar.NewReader(gz)
	for {
		header, err := inner.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(header.Name, ".mmdb") {
			continue
		}
		return io.ReadAll(inner)
	}
	return nil, fmt.Errorf("cannot find .mmdb file in archive")
}

var errNoLicence = errors.New("no maxmind license key found")

func (i *Lookup) ensureDownloaded(c app.Config, edition string) (*mmdb.Reader, error) {
	s := strings.Split(edition, "-")
	loc := c.StrOr(fmt.Sprintf("mmdb_%s", strings.ToLower(s[1])),
		fmt.Sprintf("$HOME/.$APP/maxmind/%s.mmdb", edition))
	reader, err := mmdb.Open(loc)
	if err == nil {
		// file existed, everything is fine
		return reader, err
	}
	license := c.StrOr("license", "")
	if license == "" {
		return nil, errNoLicence
	}
	raw, err := i.download(edition, license)
	if err != nil {
		return nil, fmt.Errorf("cannot download %s: %s", edition, err)
	}
	err = os.MkdirAll(path.Dir(loc), 0700)
	if err != nil {
		return nil, fmt.Errorf("cannot mkdir %s: %s", edition, err)
	}
	file, err := os.Create(loc)
	if err != nil {
		return nil, fmt.Errorf("cannot create %s: %s", edition, err)
	}
	defer file.Close()
	_, err = file.Write(raw)
	if err != nil {
		return nil, fmt.Errorf("cannot write %s: %s", edition, err)
	}
	return mmdb.FromBytes(raw)
}

type mmRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	City struct {
		Names struct {
			English string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"city"`
	Provider string `maxminddb:"autonomous_system_organization"`
	ASN      uint16 `maxminddb:"autonomous_system_number"`
}

type Info struct {
	Country  string
	City     string
	Provider string
	ASN      uint16
}

func (i Info) String() string {
	raw := []string{i.Country, i.City, i.Provider}
	return strings.Join(raw, " / ")
}

// TODO: later check for 'Iran', 'Iraq', 'Saudi Arabia',
// 'Afghanistan', 'Syria', 'Nigeria', 'Somalia'

func (i *Lookup) Available() bool {
	return i.city != nil
}

func (i *Lookup) Get(p pmux.Proxy) (info Info) {
	if !i.Available() {
		return info
	}
	// make it through request-reply channel if we're updating
	var mm mmRecord
	_ = i.asn.Lookup(p.IP(), &mm)
	info.Provider = mm.Provider
	info.ASN = mm.ASN
	_ = i.city.Lookup(p.IP(), &mm)
	info.Country = mm.Country.ISOCode
	info.City = mm.City.Names.English
	return info
}
