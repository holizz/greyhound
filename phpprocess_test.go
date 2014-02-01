package greyhound

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunPhpReturnsErrors(t *testing.T) {
	p1, _, err := runPhp("test-dir", "localhost:31524")
	defer p1.Process.Kill()
	assert.Nil(t, err)

	p2, _, err := runPhp("test-dir", "localhost:31524")
	defer p2.Process.Kill()
	assert.NotNil(t, err)
}

func TestListenOnDifferentPorts(t *testing.T) {
	ph1, err := NewPhpHandler("test-dir", 1000)
	defer ph1.Close()
	assert.Nil(t, err)

	ph2, err := NewPhpHandler("test-dir", 1000)
	defer ph2.Close()
	assert.Nil(t, err)

	assert.NotEqual(t, ph1.host, ph2.host)
}

func TestNormalRequest(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 1000)
	defer ph.Close()
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/abc.php", nil)
	assert.Nil(t, err)

	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc")
}

func TestHeaders(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 1000)
	defer ph.Close()
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/headers.php", nil)
	assert.Nil(t, err)

	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)

	assert.Equal(t, w.Code, 404)
	assert.Equal(t, w.HeaderMap["X-Golang-Is"], []string{"Awesome"})
	assert.Equal(t, w.Body.String(), "Hello from PHP\n")
}

func TestRedirects(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 1000)
	defer ph.Close()
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/redirect.php", nil)
	assert.Nil(t, err)

	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)

	assert.Equal(t, w.Code, 301)
	assert.Equal(t, w.HeaderMap["Location"], []string{"/"})
	assert.Equal(t, w.Body.String(), "")
}

func TestErrors(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 1000)
	defer ph.Close()
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/error.php", nil)
	assert.Nil(t, err)

	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)

	assert.Equal(t, w.Code, 500)
	assert.Contains(t, w.Body.String(), "Undefined variable: abc in")
	assert.Contains(t, w.Body.String(), "/error.php on line 1")
}

func TestErrorReset(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 1000)
	defer ph.Close()
	assert.Nil(t, err)

	// First request

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/multiple-errors.php", nil)
	assert.Nil(t, err)
	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)
	assert.Equal(t, w.Code, 500)

	// Second request
	w = httptest.NewRecorder()
	r, err = http.NewRequest("GET", "/abc.php", nil)
	assert.Nil(t, err)
	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc")
}

func TestLocking(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 1000)
	defer ph.Close()
	assert.Nil(t, err)

	// First request

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/wait-and-error.php", nil)
	assert.Nil(t, err)
	go func() {
		err = ph.ServeHTTP(w, r)
		assert.Nil(t, err)
		assert.Equal(t, w.Code, 500)
	}()

	// Second request
	w = httptest.NewRecorder()
	r, err = http.NewRequest("GET", "/abc.php", nil)
	assert.Nil(t, err)
	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)
	assert.Equal(t, w.Code, 200)
}

func TestTimeout(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 100)
	defer ph.Close()
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/wait-too-long.php", nil)
	assert.Nil(t, err)
	err = ph.ServeHTTP(w, r)
	assert.Nil(t, err)
	assert.Equal(t, w.Code, 500)
}
