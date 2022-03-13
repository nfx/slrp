package app

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
)

func MountSpaUI(fs fs.FS) func(*mainServer) spaFS {
	return func(server *mainServer) spaFS {
		spa := spaFS{fs}
		server.router.PathPrefix("/").Handler(spa)
		return spa
	}
}

// Single Page Application asset serving wrapper, that works both
// with embed.FS and os.DirFS
type spaFS struct {
	fs.FS
}

// findWebRoot finds a folder, where index.html is
func (sf spaFS) findWebRoot() (string, error) {
	nesting := 0
	current := ""
	for nesting <= 3 {
		file, err := sf.FS.Open(path.Clean(current))
		if err != nil {
			return "", err
		}
		rd, ok := file.(fs.ReadDirFile)
		if !ok {
			return "", fmt.Errorf("can't read dir")
		}
		files, err := rd.ReadDir(-1)
		if err != nil {
			return "", err
		}
		if len(files) == 1 && files[0].IsDir() {
			current = path.Join(current, files[0].Name())
			continue
		}
		for _, v := range files {
			if v.Name() == "index.html" {
				return current, nil
			}
		}
		nesting++
	}
	return "", fmt.Errorf("can't find index.html")
}

// Open uses unlerlying fs.FS to open file prepended by a web root prefix
func (sf spaFS) Open(name string) (fs.File, error) {
	root, err := sf.findWebRoot()
	if err != nil {
		return nil, err
	}
	file, err := sf.FS.Open(path.Join(root, name))
	if os.IsNotExist(err) {
		// fallback to React Router History API
		return sf.FS.Open(path.Join(root, "index.html"))
	}
	return file, err
}

// ServeHTTP serves static assets for itself
func (sf spaFS) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	http.FileServer(http.FS(sf)).ServeHTTP(rw, r)
}
