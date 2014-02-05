package greyhound

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FallbackHandler struct {
	dir            string
	fileServer     http.Handler
	fallbackSuffix string
	fallback       http.Handler
}

// If a request represents an extant non-directory file and that file doesn't end with fallbackSuffix, serve with an http.FileServer, otherwise use fallback.
func NewFallbackHandler(dir string, fallbackSuffix string, fallback http.Handler) (fh *FallbackHandler) {
	fh = &FallbackHandler{
		dir:            dir,
		fileServer:     http.FileServer(http.Dir(dir)),
		fallbackSuffix: fallbackSuffix,
		fallback:       fallback,
	}
	return
}

func (fh *FallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(fh.dir, r.URL.Path)


	if !strings.HasSuffix(r.URL.Path, fh.fallbackSuffix) {
		_, err := os.Stat(path)
		if !os.IsNotExist(err) {
			fh.fileServer.ServeHTTP(w, r)
			return
		}
	}

	fh.fallback.ServeHTTP(w, r)
}
