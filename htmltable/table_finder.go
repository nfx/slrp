package htmltable

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/nfx/slrp/app"

	"github.com/PuerkitoBio/goquery"
)

type Page struct {
	Tables        []*Table
	StartHeaderAt int
	ctx           context.Context
}

func New(ctx context.Context, r io.Reader) (*Page, error) {
	tc := &Page{
		ctx: ctx,
	}
	err := tc.Init(r)
	return tc, err
}

func (tc *Page) Init(r io.Reader) error {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return err
	}
	document.Find("table").Each(func(i int, s *goquery.Selection) {
		t := tc.parseTable(s)
		if t != nil {
			tc.Tables = append(tc.Tables, t)
		}
	})
	return nil
}

func NewFromString(r string) (*Page, error) {
	return New(context.Background(), strings.NewReader(r))
}

type ResponseError struct {
	Response *http.Response
	Inner    error
}

func (re *ResponseError) Error() string {
	return re.Inner.Error()
}

func NewFromHttpResponse(resp *http.Response) (*Page, error) {
	page, err := New(resp.Request.Context(), resp.Body)
	if err != nil {
		// wrap error with http response
		err = &ResponseError{resp, err}
	}
	return page, err
}

func (page *Page) Len() int {
	return len(page.Tables)
}

func intAttrOr(s *goquery.Selection, attr string, default_ int) int {
	sval, ok := s.Attr(attr)
	if !ok {
		return default_
	}

	val, err := strconv.Atoi(sval)
	if err != nil {
		return default_
	}
	return val
}

func (page *Page) parseTable(table *goquery.Selection) *Table {
	rows := table.Find("tr")
	// some strange anti-scrapping techniques may happen
	header := rows.Eq(page.StartHeaderAt)
	if page.StartHeaderAt+1 > rows.Length() {
		return nil
	}
	data := rows.Slice(page.StartHeaderAt+1, rows.Length())
	nt := &Table{}

	rowSpans := map[string]int{}
	colSpans := map[string]int{}
	// TODO: rowspans & colspans are not yet handled
	header.Find("td, th").Each(func(i int, th *goquery.Selection) {
		// alternatively we can break out early
		text := strings.Trim(th.Text(), " \r\n\t")
		rowSpans[text] = intAttrOr(th, "rowspan", 1)
		colSpans[text] = intAttrOr(th, "colspan", 1)
		nt.header = append(nt.header, text)
	})
	maxRowSpan := 1
	for _, span := range rowSpans {
		if span > maxRowSpan {
			maxRowSpan = span
		}
	}
	if maxRowSpan > 1 {
		// only supports 2 for now
		secondRow := []string{}
		header = data.Eq(0)
		data = data.Slice(1, data.Length())
		header.Find("td, th").Each(func(i int, th *goquery.Selection) {
			text := strings.Trim(th.Text(), " \r\n\t")
			secondRow = append(secondRow, text)
		})
		newHeader := []string{}
		si := 0
		for _, text := range nt.header {
			if rowSpans[text] == 2 {
				newHeader = append(newHeader, text)
				continue
			}
			if colSpans[text] > 1 {
				ci := 0
				for ci < colSpans[text] {
					newHeader = append(newHeader, text+" "+secondRow[si+ci])
					ci++
				}
				// store last pos of col
				si = si + ci
				continue
			}
			newHeader = append(newHeader, text)

		}
		nt.header = newHeader
	}
	log := app.Log.From(page.ctx)
	log.Trace().Strs("columns", nt.header).Int("count", len(nt.header)).Msg("found table")
	headerLen := len(nt.header)
	data.Each(func(i int, tr *goquery.Selection) {
		row := make([]string, headerLen)
		tr.Find("td").EachWithBreak(func(i int, td *goquery.Selection) bool {
			if i == headerLen {
				// we'll add colspan/rowspan later. maybe
				return false
			}
			row[i] = strings.Trim(td.Text(), " \r\n\t")
			return true
		})
		nt.rows = append(nt.rows, row)
	})
	return nt
}

func (page *Page) FindWithColumns(columns ...string) (*Table, error) {
	// realistic page won't have this much
	found := 0xfffffff
	for idx, table := range page.Tables {
		matchedColumns := 0
		for _, col := range columns {
			for _, header := range table.header {
				if col == header {
					matchedColumns++
				}
			}
		}
		// perform fuzzy matching of table headers
		if matchedColumns == len(columns) {
			if found < len(page.Tables) {
				// and do a best-effort error message, that is cleaner than pandas.read_html
				return nil, fmt.Errorf("more than one table matches columns `%s`: "+
					"[%d] %s and [%d] %s",
					strings.Join(columns, ", "),
					found, page.Tables[found],
					idx, page.Tables[idx],
				)
			}
			found = idx
		}
	}
	if found > len(page.Tables) {
		return nil, fmt.Errorf("cannot find table with columns: %s",
			strings.Join(columns, ", "))
	}
	return page.Tables[found], nil
}

func (page *Page) Each(a string, f func(a string) error) error {
	table, err := page.FindWithColumns(a)
	if err != nil {
		return err
	}
	offsets := map[string]int{}
	for idx, header := range table.header {
		offsets[header] = idx
	}
	for _, row := range table.rows {
		err = f(row[offsets[a]])
		if err != nil {
			return err
		}
	}
	return nil
}

func (page *Page) Each2(a, b string, f func(a, b string) error) error {
	table, err := page.FindWithColumns(a, b)
	if err != nil {
		return err
	}
	offsets := map[string]int{}
	for idx, header := range table.header {
		offsets[header] = idx
	}
	_1, _2 := offsets[a], offsets[b]
	for _, row := range table.rows {
		err = f(row[_1], row[_2])
		if err != nil {
			return err
		}
	}
	return nil
}

func (page *Page) Each3(a, b, c string, f func(a, b, c string) error) error {
	table, err := page.FindWithColumns(a, b, c)
	if err != nil {
		return err
	}
	offsets := map[string]int{}
	for idx, header := range table.header {
		offsets[header] = idx
	}
	_1, _2, _3 := offsets[a], offsets[b], offsets[c]
	for _, row := range table.rows {
		err = f(row[_1], row[_2], row[_3])
		if err != nil {
			return err
		}
	}
	return nil
}

func (page *Page) Fill(dst interface{}) error {
	vc := reflect.ValueOf(dst)
	if vc.Kind() != reflect.Chan {
		return fmt.Errorf("channel expected, got %v", dst)
	}
	elem := vc.Type().Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("channel of structs expected, got %v", elem.Name())
	}
	headers := []string{}
	fields := map[string]int{}
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		header := field.Tag.Get("header")
		if header == "" {
			continue
		}
		if field.Type.Kind() != reflect.String {
			return fmt.Errorf("only strings are supported, got %v", field.Type.Name())
		}
		fields[header] = i
		headers = append(headers, header)
	}
	table, err := page.FindWithColumns(headers...)
	if err != nil {
		return err
	}
	headerToStructField := map[int]int{}
	for idx, header := range table.header {
		field, ok := fields[header]
		if !ok {
			continue
		}
		headerToStructField[idx] = field
	}
	go func() {
		// start a goroutine in background and wait for it to be consumed
		// TODO: add context for early stop
		for _, row := range table.rows {
			item := reflect.New(elem).Elem()
			for idx, field := range headerToStructField {
				// remember, we work only with strings now
				item.Field(field).SetString(row[idx])
			}
			vc.Send(item)
		}
		vc.Close()
	}()
	return nil
}

type Table struct {
	header []string
	rows   [][]string
}

func (table *Table) String() string {
	return fmt.Sprintf("Table[%s] (%d rows)", strings.Join(table.header, ", "), len(table.rows))
}
