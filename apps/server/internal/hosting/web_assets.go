package hosting

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// LoadWebFS confines public files to a built artifact tree, including symlink resolution.
func LoadWebFS(directory string) (fs.FS, error) {
	root, e := os.OpenRoot(directory)
	if e != nil {
		return nil, fmt.Errorf("WEB_ROOT unavailable; build apps/web first: %w", e)
	}
	files := root.FS()
	index, e := fs.ReadFile(files, "index.html")
	if e == nil && len(index) == 0 {
		e = fmt.Errorf("empty index.html")
	}
	if e == nil {
		_, e = fs.ReadDir(files, "assets")
	}
	if e != nil {
		root.Close()
		return nil, fmt.Errorf("invalid WEB_ROOT; build apps/web first: %w", e)
	}
	return files, nil
}

func webUIPath(p string) bool {
	if p == "/" || p == "/login" || p == "/applications" || p == "/credentials" || p == "/device" {
		return true
	}
	return strings.HasPrefix(p, "/applications/") && idPattern.MatchString(strings.TrimPrefix(p, "/applications/"))
}
func (s *Server) webPage(w http.ResponseWriter, r *http.Request) error {
	if s.Web == nil || !webUIPath(r.URL.Path) {
		return fail(404, "not_found", "Not found")
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'")
	content, e := fs.ReadFile(s.Web, "index.html")
	if e != nil {
		return e
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	if r.Method != http.MethodHead {
		_, e = w.Write(content)
	}
	return e
}
func (s *Server) webAsset(w http.ResponseWriter, r *http.Request) error {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if s.Web == nil || !fs.ValidPath(name) || path.Clean(name) != name || strings.ContainsAny(name, "\\%") {
		return fail(404, "not_found", "Resource not found")
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") {
			return fail(404, "not_found", "Resource not found")
		}
	}
	f, e := s.Web.Open(name)
	if e != nil {
		return fail(404, "not_found", "Resource not found")
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return fail(404, "not_found", "Resource not found")
	}
	reader, ok := f.(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("Web artifact is not seekable")
	}
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeContent(w, r, name, info.ModTime(), reader)
	return nil
}
