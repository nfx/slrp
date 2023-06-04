package meta

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Meta struct {
	Tool       string
	Source     string
	TargetFile string
	Package    string
	Type       *TypeMeta
}

type state struct {
	fset            *token.FileSet
	root            string
	mod             string
	allowUnexported []string
	seen            map[string]bool
	files           []*fState
	types           map[string]*TypeMeta
}

func findGoMod(file string) (string, error) {
	dir := filepath.Dir(file)
	for {
		probe := filepath.Join(dir, "go.mod")
		_, err := os.Stat(probe)
		if err == nil {
			return probe, nil
		}
		// Stop if we have reached the root directory
		if dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("go.mod file not found")
}

func newState(filename string) (*state, error) {
	goMod, err := findGoMod(filename)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(goMod)
	if err != nil {
		return nil, err
	}
	mRE := regexp.MustCompile(`module (.*)`)
	moduleMatch := mRE.FindStringSubmatch(string(raw))
	if len(moduleMatch) != 2 {
		return nil, fmt.Errorf("no module info found in %s", goMod)
	}
	return &state{
		fset:  token.NewFileSet(),
		root:  path.Dir(goMod),
		mod:   moduleMatch[1],
		seen:  map[string]bool{},
		types: map[string]*TypeMeta{},
	}, nil
}

func (s *state) start(filename, forType string) error {
	s.allowUnexported = append(s.allowUnexported, forType)
	return s.dir(path.Dir(filename))
}

func (s *state) dir(d string) error {
	dir, err := os.Open(d)
	if err != nil {
		return err
	}
	defer dir.Close()
	files, err := dir.ReadDir(0)
	if err != nil {
		return err
	}
	for _, e := range files {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		filename := path.Join(d, e.Name())
		if strings.HasSuffix(filename, "_test.go") {
			// not interested in types defined in test for now
			continue
		}
		if s.seen[filename] {
			continue
		}
		f, err := s.parse(filename)
		if err != nil {
			return err
		}
		s.seen[filename] = true
		if len(f.types) == 0 {
			// boring file, no exported types ...
			continue
		}
		s.files = append(s.files, f)
	}
	return nil
}

type fState struct {
	s          *state
	filename   string
	pkg        string
	pkgAliases map[string]string
	types      map[string]TypeMeta
}

func (s *state) parse(filename string) (*fState, error) {
	file, err := parser.ParseFile(s.fset, filename, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	dir := path.Dir(filename)
	pkg := strings.Replace(dir, s.root, s.mod, 1)
	f := &fState{
		s:          s,
		filename:   filename,
		pkg:        pkg,
		pkgAliases: map[string]string{},
		types:      map[string]TypeMeta{},
	}
	for _, v := range file.Decls {
		err = f.decl(v)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
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

type TypeMeta struct {
	Name   string
	Fields []FieldMeta2

	methods map[string]bool
	f       *fState
}

type FieldMeta2 struct {
	Name  string
	Type  *TypeRef
	Facet string
}

type TypeRef struct {
	Pkg       string
	Name      string
	IsArray   bool
	IsPointer bool
	MapKey    *TypeRef
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
	t := TypeMeta{
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
			t.Fields = append(t.Fields, FieldMeta2{
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

func (f *fState) typeRef(e ast.Expr) *TypeRef {
	switch x := e.(type) {
	case *ast.Ident:
		return &TypeRef{
			Name: f.ident(x),
		}
	case *ast.SelectorExpr:
		return &TypeRef{
			Pkg:  f.typeRef(x.X).Name,
			Name: f.ident(x.Sel),
		}
	case *ast.ArrayType:
		y := f.typeRef(x.Elt)
		if y == nil {
			return nil
		}
		return &TypeRef{
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
		return &TypeRef{
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
		return &TypeRef{
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

func Parse(filename, pkg, forType string) (*Meta, error) {
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
	return &Meta{
		Tool:       strings.Join(os.Args, " "),
		Source:     forType,
		TargetFile: targetName,
		Package:    pkg,
		Type:       t,
	}, nil
}
