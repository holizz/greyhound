package greyhound

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestListenOnDifferentPorts(t *testing.T) {
	ph1, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph1.Close()
	assert.Nil(t, err)

	ph2, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph2.Close()
	assert.Nil(t, err)

	assert.NotEqual(t, ph1.host, ph2.host)
}

func TestNormalRequest(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/abc.php")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc")
}

func TestHeaders(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/headers.php")

	assert.Equal(t, w.Code, 404)
	assert.Equal(t, w.HeaderMap["X-Golang-Is"], []string{"Awesome"})
	assert.Equal(t, w.Body.String(), "Hello from PHP\n")
}

func TestRedirects(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/redirect.php")

	assert.Equal(t, w.Code, 301)
	assert.Equal(t, w.HeaderMap["Location"], []string{"/"})
	assert.Equal(t, w.Body.String(), "")
}

func TestErrors(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/error.php")

	assert.Equal(t, w.Code, 500)
	assert.Contains(t, w.Body.String(), "PHP Notice:  Undefined variable: abc in")
	assert.Contains(t, w.Body.String(), "/error.php on line 1")
}

func TestTimeout(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", 100, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/wait-too-long.php")
	assert.Equal(t, w.Code, 500)
}

func TestErrorIgnoring(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Millisecond*100, []string{}, []string{"/error.php on line 1"})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/error.php")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "123 ")
}

func TestFatalErrorIgnoring(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Millisecond*100, []string{}, []string{"/"})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/fatal-error.php")

	assert.Equal(t, w.Code, 500)
	assert.Contains(t, w.Body.String(), "PHP Fatal error:  Call to undefined function flub() in")
}

func TestPassingFlagsToPhp(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Millisecond*100, []string{"-d", "error_reporting=E_STRICT"}, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	w := get(t, ph, "/error.php")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "123 ")
}

func TestFailOnSecondUse(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	// First request
	w := get(t, ph, "/abc.php")
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc")

	// Second request
	w = get(t, ph, "/abc.php")
	assert.Equal(t, w.Code, 500)
	assert.Contains(t, w.Body.String(), "cannot be used twice")
}
