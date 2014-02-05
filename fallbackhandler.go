package greyhound

import (
	"net/http"
)

type FallbackHandler struct {
	fileServer http.Handler
	fallbackSuffix string
	fallback http.Handler
}

// If a request represents an extant non-directory file and that file doesn't end with fallbackSuffix, serve with an http.FileServer, otherwise use fallback.
func NewFallbackHandler(dir string, fallbackSuffix string, fallback http.Handler) (fh *FallbackHandler) {
	return
}

func (fh *FallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
