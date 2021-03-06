package greyhound

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// If a request represents a file that doesn't end with fallbackSuffix, serve it with http.FileHandler, otherwise serve it with fallback.
type FallbackHandler struct {
	dir            string
	fileServer     http.Handler
	fallbackSuffix string
	fallback       http.Handler
}

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
		fileInfo, err := os.Stat(path)
		if !os.IsNotExist(err) && !fileInfo.IsDir() {
			fh.fileServer.ServeHTTP(w, r)
			return
		}
	}

	fh.fallback.ServeHTTP(w, r)
}
