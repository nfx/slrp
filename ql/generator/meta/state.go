package meta

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type state struct {
	fset            *token.FileSet
	root            string
	mod             string
	allowUnexported []string
	seen            map[string]bool
	files           []*fState
	types           map[string]*Type
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
	return "", fmt.Errorf("go.mod file not found for %s", file)
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
		types: map[string]*Type{},
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
		types:      map[string]Type{},
	}
	for _, v := range file.Decls {
		err = f.decl(v)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}
