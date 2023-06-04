package meta

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode"
)

type fState struct {
	s          *state
	filename   string
	pkg        string
	pkgAliases map[string]string
	types      map[string]Type
}

func (f *fState) decl(d ast.Decl) error {
	switch x := d.(type) {
	case *ast.GenDecl:
		return f.genDecl(x)
	case *ast.FuncDecl:
		return f.funcDecl(x)
	default:
		return nil
	}
}

func (f *fState) funcDecl(fu *ast.FuncDecl) error {
	funcName := f.ident(fu.Name)
	if !f.isExported(funcName) {
		// private member
		return nil
	}
	if fu.Recv == nil {
		// not a method
		return nil
	}
	for _, v := range fu.Recv.List {
		typeRef := f.typeRef(v.Type)
		// shortcut: methods must be defined after type declaration,
		// even though Go allows for the opposite cases. This assumes
		// that f.typeSpec() was called *before* f.funcDecl()
		t, defined := f.types[typeRef.Name]
		if !defined {
			continue
		}
		// here we only store the names of exported method names,
		// NOT their proper interfaces (which we technically can),
		// which is sufficient to know if type has a String() or not.
		//
		// once we allow nested field lookup in the query engine,
		// we could store a bit more detailed field signature,
		// so that we use method return values for comparisons.
		t.methods[funcName] = true
	}
	return nil
}

func (f *fState) genDecl(gd *ast.GenDecl) error {
	for _, s := range gd.Specs {
		err := f.spec(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *fState) basicLit(b *ast.BasicLit) string {
	if b == nil {
		return ""
	}
	return strings.Trim(b.Value, "\"")
}

func (f *fState) ident(i *ast.Ident) string {
	if i == nil {
		return ""
	}
	return i.Name
}

func (f *fState) allowType(n string) bool {
	for _, v := range f.s.allowUnexported {
		if v == n {
			return true
		}
	}
	return f.isExported(n)
}

func (f *fState) typeSpec(s *ast.TypeSpec) error {
	name := f.ident(s.Name)
	if !f.allowType(name) {
		return nil
	}
	_, defined := f.types[name]
	if defined {
		// type is defined already?..
		return nil
	}
	t := Type{
		Name:    name,
		methods: map[string]bool{},
		f:       f,
	}
	switch x := s.Type.(type) {
	case *ast.StructType:
		for _, v := range x.Fields.List {
			fieldName := f.fieldName(v)
			if !f.isExported(fieldName) {
				continue
			}
			t.Fields = append(t.Fields, Field{
				Name:  fieldName,
				Type:  f.typeRef(v.Type),
				Facet: f.facet(v.Tag),
			})
		}
	case *ast.InterfaceType:
		// interfaces are not searchable
		return nil
	case *ast.FuncType:
		// function type aliases are definitely not searchable
		return nil
	}
	f.types[name] = t
	// we hope to match it by Stringer or something
	fqtn := fmt.Sprintf("%s.%s", f.pkg, name)
	f.s.types[fqtn] = &t
	return nil
}

func (f *fState) facet(t *ast.BasicLit) string {
	if t == nil {
		return ""
	}
	for _, v := range strings.Split(
		strings.Trim(t.Value, "`"), " ") {
		split := strings.Split(v, ":")
		if len(split) != 2 {
			continue
		}
		// we ignore any other tags than "facet"
		if split[0] != "facet" {
			continue
		}
		return strings.Trim(split[1], `"`)
	}
	return ""
}

func (f *fState) typeRef(e ast.Expr) *Ref {
	switch x := e.(type) {
	case *ast.Ident:
		return &Ref{
			Name: f.ident(x),
		}
	case *ast.SelectorExpr:
		return &Ref{
			Pkg:  f.typeRef(x.X).Name,
			Name: f.ident(x.Sel),
		}
	case *ast.ArrayType:
		y := f.typeRef(x.Elt)
		if y == nil {
			return nil
		}
		return &Ref{
			IsArray:   true,
			Pkg:       y.Pkg,
			Name:      y.Name,
			IsPointer: y.IsPointer,
			MapKey:    y.MapKey,
		}
	case *ast.StarExpr:
		y := f.typeRef(x.X)
		if y == nil {
			return nil
		}
		return &Ref{
			IsPointer: true,
			Pkg:       y.Pkg,
			Name:      y.Name,
			IsArray:   y.IsArray,
			MapKey:    y.MapKey,
		}
	case *ast.MapType:
		v := f.typeRef(x.Value)
		if v == nil {
			return nil
		}
		return &Ref{
			Pkg:       v.Pkg,
			Name:      v.Name,
			IsPointer: v.IsPointer,
			IsArray:   v.IsArray,
			MapKey:    f.typeRef(x.Key),
		}
	case *ast.FuncType:
		// let's ignore it...
		return nil
	default:
		return nil
	}
}

func (f *fState) isExported(n string) bool {
	return n != "" && unicode.IsUpper(rune(n[0]))
}

func (f *fState) fieldName(af *ast.Field) string {
	names := []string{}
	for _, v := range af.Names {
		names = append(names, f.ident(v))
	}
	return strings.Join(names, ".")
}

func (f *fState) importSpec(s *ast.ImportSpec) error {
	path := f.basicLit(s.Path)
	if !strings.HasPrefix(path, f.s.mod) {
		// not interested in files outside our module
		return nil
	}
	name := f.ident(s.Name)
	if name == "" {
		split := strings.Split(path, "/")
		name = split[len(split)-1]
	}
	f.pkgAliases[name] = path
	pkgDir := strings.Replace(path, f.s.mod, f.s.root, 1)
	return f.s.dir(pkgDir)
}

func (f *fState) spec(s ast.Spec) error {
	switch x := s.(type) {
	case *ast.ImportSpec:
		return f.importSpec(x)
	case *ast.TypeSpec:
		return f.typeSpec(x)
	case *ast.ValueSpec:
		return nil
	default:
		return nil
	}
}
