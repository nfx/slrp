package ql

import (
	"reflect"
	"testing"
)

type x struct {
	First, Second, Third int
	Fifth                string
}

var fixture = []x{
	{1, 6, 10, "aabb"},
	{1, 5, 9, "ccdd"},
	{1, 5, 8, "eeff"},
	{2, 4, 7, "gghh"},
	{2, 3, 6, "iijj"},
	{2, 3, 5, "hhii"},
	{3, 2, 4, "ffgg"},
	{3, 1, 3, "ddee"},
	{3, 1, 2, "bbcc"},
}

type tt struct {
	query string
	want  *Query[x]
	err   error
}

func TestParse(t *testing.T) {
	tests := []tt{
		// time.ParseDuration("5h30m40s")
		// {"Second > 22", nil, nil},
		// {"Second > 22 ORDER BY First, Second DESC LIMIT 2", nil, nil},
		{"Fifth~a OR Fifth~d ORDER BY First, Second DESC LIMIT 2", nil, nil},
		//{"foo:bar AND NOT (bar=\"baz\" OR foo ~ 1) ORDER BY foo, bar DESC", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			query, err := Parse[x](tt.query)

			var result []x
			err = query.Apply(&fixture, &result)
			if err != tt.err {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.want)
				return
			}
			if !reflect.DeepEqual(query, tt.want) {
				t.Errorf("Parse() = %+v, want %v", query, tt.want)
			}
		})
	}
}
