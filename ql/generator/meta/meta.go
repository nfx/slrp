package meta

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type Dataset struct {
	Tool       string
	TargetFile string
	Type       *Type
}

type Type struct {
	Name   string
	Fields []Field

	methods map[string]bool
	f       *fState
}

func (t *Type) Package() string {
	return path.Base(t.f.pkg)
}

type Field struct {
	Name  string
	Ref   *Ref
	Of    *Type
	Facet string
}

func (f Field) AbstractType() string {
	t := f.Ref
	if t == nil {
		return "unknown"
	}
	return t.AbstractType()
}

func Parse(filename, forType string) (*Dataset, error) {
	c, err := newState(filename)
	if err != nil {
		return nil, err
	}
	err = c.start(filename, forType)
	if err != nil {
		return nil, err
	}
	pkg := strings.ReplaceAll(path.Dir(filename), c.root, c.mod)
	fqtn := fmt.Sprintf("%s.%s", pkg, forType)
	t, ok := c.types[fqtn]
	if !ok {
		return nil, fmt.Errorf("no type found: %s", fqtn)
	}
	targetName := fmt.Sprintf("%s_dataset.go", strings.ToLower(forType))
	if strings.HasSuffix(filename, "_test.go") {
		// special case for package under test, for simplicity reasons.
		targetName = targetName[:len(targetName)-3] + "_test.go"
	}
	return &Dataset{
		Tool:       strings.Join(os.Args, " "),
		TargetFile: targetName,
		Type:       t,
	}, nil
}
