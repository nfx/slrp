package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/nfx/slrp/ql/generator/meta"
)

const tmpl = `// Code generated by go run github.com/nfx/slrp/ql/generator/main.go Foo. DO NOT EDIT.
package {{.Package}}

{{if ev }}
import (
	"github.com/nfx/slrp/ql/eval"
)
{{end}}

type {{.Source}}Dataset []{{.Source}}

func (d {{.Source}}Dataset) Query(query string) (*{{ev}}QueryResult[{{.Source}}], error) {
	return (&{{ev}}Dataset[{{.Source}}]{
		Source: d,
		Accessors: {{ev}}Accessors{
			{{range .Fields -}}
				{{FieldTemplate "accessor" .}}
			{{end}}
		},
		Sorters: {{ev}}Sorters[{{.Source}}]{
			{{range .Fields -}}
				"{{.Name}}": {Asc: d.sortAsc{{.Name}}, Desc: d.sortDesc{{.Name}}},
			{{end}}
		},
		Facets: {{ev}}FacetRetrievers[{{.Source}}]{
			{{range .Fields -}}
				{{FieldTemplate "facet" .}}
			{{- end}}
		},
	}).Query(query)
}

{{range .Fields}}
	{{FieldTemplate "methods" .}}
{{end}}

{{define "facet-string"}}
	{{if .Tags.facet }}
		{{ev}}StringFacet{
			Getter: d.get{{.Name}}, 
			Field: "{{.Name}}", 
			Name: "{{.Tags.facet}}",
		},
	{{end}}
{{end}}
{{define "facet-int"}}{{end}}
{{define "facet-bool"}}{{end}}

{{define "accessor-number"}}
	"{{.Name}}": {{ev}}NumberGetter{"{{.Name}}", d.get{{.Name}}},
{{end}}

{{define "accessor-string"}}
	"{{.Name}}": {{ev}}StringGetter{"{{.Name}}", d.get{{.Name}}},
{{end}}

{{define "accessor-bool"}}
	"{{.Name}}": {{ev}}BooleanGetter{"{{.Name}}", d.get{{.Name}}},
{{end}}

{{define "methods-number"}}
	func (d {{.Of.Name}}Dataset) get{{.Name}}(record int) float64 {
		{{if eq .Ref "time.Time" -}}
		return float64(d[record].Unix())
		{{- else if eq .Ref "time.Duration" -}}
		return float64(d[record])
		{{- else -}}
		return float64(d[record].{{.Name}})
		{{- end}}
	}
	{{template "sort" .}}
{{end}}

{{define "methods-string"}}
	func (d {{.Of.Name}}Dataset) get{{.Name}}(record int) string {
		return d[record].{{.Name}}{{if .Ref.IsStringer}}.String(){{end}}
	}
	{{template "sort" .}}
{{end}}

{{define "methods-bool"}}
	func (d {{.Of.Name}}Dataset) get{{.Name}}(record int) bool {
		return d[record].{{.Name}}
	}

	func (_ {{.Of.Name}}Dataset) sortAsc{{.Name}}(left, right {{.Of.Name}}) bool {
		return left.{{.Name}} == right.{{.Name}}
	}

	func (_ {{.Of.Name}}Dataset) sortDesc{{.Name}}(left, right {{.Of.Name}}) bool {
		return left.{{.Name}} != right.{{.Name}}
	}
{{end}}

{{define "sort"}}
	func (_ {{.Of.Name}}Dataset) sortAsc{{.Name}}(left, right {{.Of.Name}}) bool {
		return left.get{{.Name}}() < right.get{{.Name}}()
	}

	func (_ {{.Of.Name}}Dataset) sortDesc{{.Name}}(left, right {{.Of.Name}}) bool {
		return left.get{{.Name}}() > right.get{{.Name}}()
	}
{{end}}
`

func main() {
	goFile := os.Getenv("GOFILE")
	goPkg := os.Getenv("GOPACKAGE")
	goType := os.Args[1]
	ds, err := meta.Parse(goFile, goPkg, goType)
	if err != nil {
		panic(err)
	}
	t := template.New("code")
	t.Funcs(template.FuncMap{
		"ev": func() string {
			// special case for package under test, for simplicity reasons.
			if goPkg == "eval" {
				return ""
			}
			return "eval."
		},
		"FieldTemplate": func(prefix string, field *meta.Field) (string, error) {
			buf := bytes.NewBuffer([]byte{})
			err = t.ExecuteTemplate(buf, fmt.Sprintf("%s-%s", prefix, field.AbstractType()), field)
			return strings.TrimSpace(buf.String()), err
		},
	}).Parse(tmpl)
	dst, err := os.OpenFile(ds.TargetFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	defer dst.Close()
	err = t.Execute(dst, ds)
	if err != nil {
		panic(err)
	}
	err = exec.Command("go", "fmt", ds.TargetFile).Run()
	if err != nil {
		panic(err)
	}
}
