package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"
	"text/template"
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

{{define "accessor-int"}}
	"{{.Name}}": {{ev}}NumberGetter{"{{.Name}}", d.get{{.Name}}},
{{end}}

{{define "accessor-string"}}
	"{{.Name}}": {{ev}}StringGetter{"{{.Name}}", d.get{{.Name}}},
{{end}}

{{define "accessor-bool"}}
	"{{.Name}}": {{ev}}BooleanGetter{"{{.Name}}", d.get{{.Name}}},
{{end}}

{{define "methods-int"}}
	func (d {{.Source}}Dataset) get{{.Name}}(record int) float64 {
		return float64(d[record].{{.Name}})
	}
	{{template "sort" .}}
{{end}}

{{define "methods-string"}}
	func (d {{.Source}}Dataset) get{{.Name}}(record int) string {
		return d[record].{{.Name}}
	}
	{{template "sort" .}}
{{end}}

{{define "methods-bool"}}
	func (d {{.Source}}Dataset) get{{.Name}}(record int) bool {
		return d[record].{{.Name}}
	}

	func (_ {{.Source}}Dataset) sortAsc{{.Name}}(left, right {{.Source}}) bool {
		return left.{{.Name}} == right.{{.Name}}
	}

	func (_ {{.Source}}Dataset) sortDesc{{.Name}}(left, right {{.Source}}) bool {
		return left.{{.Name}} != right.{{.Name}}
	}
{{end}}

{{define "sort"}}
	func (_ {{.Source}}Dataset) sortAsc{{.Name}}(left, right {{.Source}}) bool {
		return left.{{.Name}} < right.{{.Name}}
	}

	func (_ {{.Source}}Dataset) sortDesc{{.Name}}(left, right {{.Source}}) bool {
		return left.{{.Name}} > right.{{.Name}}
	}
{{end}}
`

// GOLINE=9

type FieldMeta struct {
	Source string
	Name   string
	Type   string
	Tags   map[string]string
}

type Meta struct {
	Tool       string
	Source     string
	TargetFile string
	Package    string
	Fields     []FieldMeta
}

func prepare(filename, forType string) (*Meta, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	for _, v := range file.Decls {
		keep := ast.FilterDecl(v, func(s string) bool {
			return s == forType
		})
		if !keep {
			continue
		}
		metas := []FieldMeta{}
		ast.Inspect(v, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Field:
				tags := map[string]string{}
				if x.Tag != nil {
					for _, v := range strings.Split(
						strings.Trim(x.Tag.Value, "`"), " ") {
						split := strings.Split(v, ":")
						if len(split) != 2 {
							continue
						}
						tags[split[0]] = strings.Trim(split[1], `"`)
					}
				}
				switch t := x.Type.(type) {
				case *ast.Ident:
					fieldType := t.Name
					for _, f := range x.Names {
						if !f.IsExported() {
							continue
						}
						metas = append(metas, FieldMeta{
							Source: forType,
							Name:   f.Name,
							Type:   fieldType,
							Tags:   tags,
						})
					}
				case *ast.SelectorExpr:
					ast.Print(fset, x)
					panic(fmt.Errorf("expr: %s, selector: %s", t.X, t.Sel))
					// TODO: parse import
				default:
					panic(fmt.Errorf("dunno: %v (%T)", x.Type, x.Type))
				}
			}
			return true
		})
		targetName := fmt.Sprintf("%s_dataset.go", strings.ToLower(forType))
		if strings.HasSuffix(filename, "_test.go") {
			// special case for package under test, for simplicity reasons.
			targetName = targetName[:len(targetName)-3] + "_test.go"
		}
		return &Meta{
			Tool:       strings.Join(os.Args, " "),
			Source:     forType,
			TargetFile: targetName,
			Package:    os.Getenv("GOPACKAGE"),
			Fields:     metas,
		}, nil
	}
	return nil, fmt.Errorf("no type found: %s", forType)
}

func main() {
	goFile := os.Getenv("GOFILE")
	goType := os.Args[1]
	meta, err := prepare(goFile, goType)
	if err != nil {
		panic(err)
	}
	t := template.New("code")
	t.Funcs(template.FuncMap{
		"ev": func() string {
			// special case for package under test, for simplicity reasons.
			if os.Getenv("GOPACKAGE") == "eval" {
				return ""
			}
			return "eval."
		},
		"FieldTemplate": func(prefix string, field FieldMeta) (string, error) {
			buf := bytes.NewBuffer([]byte{})
			err = t.ExecuteTemplate(buf, fmt.Sprintf("%s-%s", prefix, field.Type), field)
			return strings.TrimSpace(buf.String()), err
		},
	}).Parse(tmpl)
	dst, err := os.OpenFile(meta.TargetFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	defer dst.Close()
	err = t.Execute(dst, meta)
	if err != nil {
		panic(err)
	}
	err = exec.Command("go", "fmt", meta.TargetFile).Run()
	if err != nil {
		panic(err)
	}
}
