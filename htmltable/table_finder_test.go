package htmltable

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/assert"
)

const fixture = `<body>
<h1>foo</h2>
<table>
	<tr><td>a</td><td>b</td></tr>
	<tr><td> 1 </td><td>2</td></tr>
	<tr><td>3  </td><td>4   </td></tr>
</table>
<h1>bar</h2>
<table>
	<tr><th>b</th><th>c</th><th>d</th></tr>
	<tr><td>1</td><td>2</td><td>5</td></tr>
	<tr><td>3</td><td>4</td><td>6</td></tr>
</table>
</body>`

func TestInitFails(t *testing.T) {
	prev := goqueryNewDocumentFromReader
	t.Cleanup(func() {
		goqueryNewDocumentFromReader = prev
	})
	goqueryNewDocumentFromReader = func(r io.Reader) (*goquery.Document, error) {
		return nil, fmt.Errorf("nope")
	}
	_, err := New(context.Background(), strings.NewReader(".."))

	assert.EqualError(t, err, "nope")
}

func TestNewFromHttpResponseError(t *testing.T) {
	prev := goqueryNewDocumentFromReader
	t.Cleanup(func() {
		goqueryNewDocumentFromReader = prev
	})
	goqueryNewDocumentFromReader = func(r io.Reader) (*goquery.Document, error) {
		return nil, fmt.Errorf("nope")
	}
	_, err := NewFromResponse(&http.Response{
		Request: &http.Request{},
	})
	assert.EqualError(t, err, "nope")
}

func TestRealPageFound(t *testing.T) {
	wiki, err := http.Get("https://en.wikipedia.org/wiki/List_of_S%26P_500_companies")
	assert.NoError(t, err)
	page, err := NewFromResponse(wiki)
	assert.NoError(t, err)
	snp, err := page.FindWithColumns("Symbol", "Security", "CIK")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(snp.rows), 500)
}

func TestRealPageFound_BasicRowColSpans(t *testing.T) {
	wiki, err := http.Get("https://en.wikipedia.org/wiki/List_of_S%26P_500_companies")
	assert.NoError(t, err)
	page, err := NewFromResponse(wiki)
	assert.NoError(t, err)
	snp, err := page.FindWithColumns("Date", "Added Ticker", "Removed Ticker")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(snp.rows), 250)
}

func TestFindsAllTables(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	assert.Equal(t, page.Len(), 2)
}

func TestFindsTableByColumnNames(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)

	cd, err := page.FindWithColumns("c", "d")
	assert.NoError(t, err)
	assert.Len(t, cd.rows, 2)
}

func TestEach(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each("a", func(a string) error {
		log.Printf("[INFO] %s", a)
		return nil
	})
	assert.NoError(t, err)
}

func TestEachFails(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each("a", func(a string) error {
		return fmt.Errorf("nope")
	})
	assert.EqualError(t, err, "nope")
}

func TestEachFailsNoCols(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each("x", func(a string) error {
		return nil
	})
	assert.EqualError(t, err, "cannot find table with columns: x")
}

func TestEach2(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each2("b", "c", func(b, c string) error {
		log.Printf("[INFO] %s %s", b, c)
		return nil
	})
	assert.NoError(t, err)
}

func TestEach2Fails(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each2("b", "c", func(b, c string) error {
		return fmt.Errorf("nope")
	})
	assert.EqualError(t, err, "nope")
}

func TestEach2FailsNoCols(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each2("x", "y", func(b, c string) error {
		return nil
	})
	assert.EqualError(t, err, "cannot find table with columns: x, y")
}

func TestEach3(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each3("b", "c", "d", func(b, c, d string) error {
		log.Printf("[INFO] %s %s %s", b, c, d)
		return nil
	})
	assert.NoError(t, err)
}

func TestEach3Fails(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each3("b", "c", "d", func(b, c, d string) error {
		return fmt.Errorf("nope")
	})
	assert.EqualError(t, err, "nope")
}

func TestEach3FailsNoCols(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each3("x", "y", "z", func(b, c, d string) error {
		return nil
	})
	assert.EqualError(t, err, "cannot find table with columns: x, y, z")
}

func TestMoreThanOneTableFoundErrors(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)

	_, err = page.FindWithColumns("b")
	assert.Error(t, err)
}

func TestNoTablesFoundErrors(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)

	_, err = page.FindWithColumns("z")
	assert.Error(t, err)
}
