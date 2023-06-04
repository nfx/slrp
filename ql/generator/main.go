package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/nfx/slrp/ql/generator/meta"
)

//go:embed dataset.go.tmpl
var tmpl string

func main() {
	// Environment variables:
	//
	// PWD - directory, where the file with //go:generate is placed
	// GOFILE - basename, path.Join(os.Getenv("PWD"), os.Getenv("GOFILE")) to work properly
	// GOLINE - where the //go:generate comment is placed
	// GOPACKAGE - basename of the package, need to resolve go.mod for the full package name
	filename := path.Join(os.Getenv("PWD"), os.Getenv("GOFILE"))
	ds, err := meta.Parse(filename, os.Args[1])
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
