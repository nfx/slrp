package htmltable

import (
	"fmt"
	"net/http"
	"testing"

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

func TestRealPageFound(t *testing.T) {
	wiki, err := http.Get("https://en.wikipedia.org/wiki/List_of_S%26P_500_companies")
	assert.NoError(t, err)
	page, err := NewFromHttpResponse(wiki)
	assert.NoError(t, err)
	snp, err := page.FindWithColumns("Symbol", "Security", "CIK")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(snp.rows), 500)
}

func TestRealPageFound_BasicRowColSpans(t *testing.T) {
	wiki, err := http.Get("https://en.wikipedia.org/wiki/List_of_S%26P_500_companies")
	assert.NoError(t, err)
	page, err := NewFromHttpResponse(wiki)
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

func TestToStructSlice(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	type nice struct {
		C string `header:"c"`
		D string `header:"d"`
	}
	out := make(chan nice)
	if err = page.Fill(out); err != nil {
		for n := range out {
			fmt.Println(n)
		}
	}
}

func TestEach3(t *testing.T) {
	page, err := NewFromString(fixture)
	assert.NoError(t, err)
	err = page.Each3("b", "c", "d", func(b, c, d string) {
		log.Printf("[INFO] %s %s %s", b, c, d)
	})
	assert.NoError(t, err)
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
