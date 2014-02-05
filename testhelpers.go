package greyhound

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func get(t *testing.T, h http.Handler, uri string) (w *httptest.ResponseRecorder) {
	w = httptest.NewRecorder()
	r, err := http.NewRequest("GET", uri, nil)
	assert.Nil(t, err)

	h.ServeHTTP(w, r)

	return
}

