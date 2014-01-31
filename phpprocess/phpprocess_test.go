package phpprocess

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunPhpReturnsErrors(t *testing.T) {
	p1, err := runPhp("test-dir", "localhost:31524")
	defer p1.Kill()
	assert.Nil(t, err)

	p2, err := runPhp("test-dir", "localhost:31524")
	defer p2.Kill()
	assert.NotNil(t, err)
}

func TestListenOnDifferentPorts(t *testing.T) {
	ph1, err := NewPhpProcess("test-dir")
	defer ph1.Close()
	assert.Nil(t, err)

	ph2, err := NewPhpProcess("test-dir")
	defer ph2.Close()
	assert.Nil(t, err)

	assert.NotEqual(t, ph1.host, ph2.host)
}

func TestNormalRequest(t *testing.T) {
	ph, err := NewPhpProcess("test-dir")
	defer ph.Close()
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/abc.php", nil)
	assert.Nil(t, err)

	err = ph.MakeRequest(w, r)
	assert.Nil(t, err)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc123")
}

func TestHeaders(t *testing.T) {
	ph, err := NewPhpProcess("test-dir")
	defer ph.Close()
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/headers.php", nil)
	assert.Nil(t, err)

	err = ph.MakeRequest(w, r)
	assert.Nil(t, err)

	assert.Equal(t, w.Code, 404)
	assert.Equal(t, w.HeaderMap["X-Golang-Is"], "Awesome")
	assert.Equal(t, w.Body.String(), "Hello from PHP")
}
