package meta

type Ref struct {
	Pkg       string
	Name      string
	IsArray   bool
	IsPointer bool
	MapKey    *Ref
	f         *fState
}

func (r *Ref) String() string {
	name := r.Name
	if r.Pkg != "" {
		name = r.Pkg + "." + name
	}
	if r.IsArray {
		name = "[]" + name
	}
	if r.IsPointer {
		name = "*" + name
	}
	if r.MapKey != nil {
		name = "map[" + r.MapKey.String() + "]" + name
	}
	return name
}

func (r *Ref) IsStringer() bool {
	ourPkg, ok := r.f.pkgAliases[r.Pkg]
	if !ok {
		return false
	}
	fqtn := ourPkg + "." + r.Name
	ourType, ok := r.f.s.types[fqtn]
	// currently we only support stringy custom types
	return ok && ourType.methods["String"]
}

var coversions = []struct {
	pkg, name, target string
}{
	{"time", "Time", "number"},
	{"time", "Duration", "number"},
	{"", "bool", "bool"},
	{"", "int", "number"},
	{"", "int16", "number"},
	{"", "int32", "number"},
	{"", "int64", "number"},
	{"", "uint", "number"},
	{"", "uint32", "number"},
	{"", "uint16", "number"},
	{"", "string", "string"},
}

// abstract type of the data behind the field
func (r *Ref) AbstractType() string {
	if r.IsStringer() {
		return "string"
	}
	for _, c := range coversions {
		if c.pkg == r.Pkg && c.name == r.Name {
			return c.target
		}
	}
	return "unknown"
}
