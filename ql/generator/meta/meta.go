package meta

import (
	"fmt"
	"os"
	"strings"
)

type Dataset struct {
	Tool       string
	Source     string
	TargetFile string
	Package    string
	Type       *Type
}

type Type struct {
	Name   string
	Fields []Field

	methods map[string]bool
	f       *fState
}

type Field struct {
	Name  string
	Type  *Ref
	Facet string
}

type Ref struct {
	Pkg       string
	Name      string
	IsArray   bool
	IsPointer bool
	MapKey    *Ref
}

func Parse(filename, pkg, forType string) (*Dataset, error) {
	c, err := newState(filename)
	if err != nil {
		return nil, err
	}
	err = c.start(filename, forType)
	if err != nil {
		return nil, err
	}
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
		Source:     forType,
		TargetFile: targetName,
		Package:    pkg,
		Type:       t,
	}, nil
}
