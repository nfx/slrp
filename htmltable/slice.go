package htmltable

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

func NewSlice[T any](ctx context.Context, r io.Reader) ([]T, error) {
	f := &feeder[T]{
		Page: Page{ctx: ctx},
	}
	f.Init(r)
	return f.slice()
}

func NewSliceFromString[T any](in string) ([]T, error) {
	return NewSlice[T](context.Background(), strings.NewReader(in))
}

func NewSliceFromResponse[T any](resp *http.Response) ([]T, error) {
	out, err := NewSlice[T](resp.Request.Context(), resp.Body)
	if err != nil {
		// wrap error with http response
		return nil, &ResponseError{resp, err}
	}
	return out, nil
}

func NewSliceFromURL[T any](url string) ([]T, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	return NewSliceFromResponse[T](resp)
}

type feeder[T any] struct {
	Page
	dummy T
}

func (f *feeder[T]) headers() ([]string, map[string]int, error) {
	dt := reflect.ValueOf(f.dummy)
	elem := dt.Type()
	headers := []string{}
	fields := map[string]int{}
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		header := field.Tag.Get("header")
		if header == "" {
			continue
		}
		if field.Type.Kind() != reflect.String {
			return nil, nil, fmt.Errorf("only strings are supported, %s is %v",
				field.Name, field.Type.Name())
		}
		fields[header] = i
		headers = append(headers, header)
	}
	return headers, fields, nil
}

func (f *feeder[T]) table() (*tableData, map[int]int, error) {
	headers, fields, err := f.headers()
	if err != nil {
		return nil, nil, err
	}
	table, err := f.FindWithColumns(headers...)
	if err != nil {
		return nil, nil, err
	}
	mapping := map[int]int{}
	for idx, header := range table.header {
		field, ok := fields[header]
		if !ok {
			continue
		}
		mapping[idx] = field
	}
	return table, mapping, nil
}

func (f *feeder[T]) slice() ([]T, error) {
	table, mapping, err := f.table()
	if err != nil {
		return nil, err
	}
	dummy := reflect.ValueOf(f.dummy)
	dt := dummy.Type()
	sliceValue := reflect.MakeSlice(reflect.SliceOf(dt),
		len(table.rows), len(table.rows))
	for rowIdx, row := range table.rows {
		item := sliceValue.Index(rowIdx)
		for idx, field := range mapping {
			// remember, we work only with strings now
			item.Field(field).SetString(row[idx])
		}
	}
	return sliceValue.Interface().([]T), nil
}
