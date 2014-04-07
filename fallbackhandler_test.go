package greyhound

import (
	"testing"
	"time"

	"github.com/codegangsta/martini"
	"github.com/stretchr/testify/assert"
)

func TestStatic(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".phpx")
	m := martini.Classic()
	m.Handlers(fh.ServeHTTP, ph.ServeHTTP)

	w := get(t, m, "/abc.php")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "<?php echo 'abc' ?>\n")
}

func TestStaticAndPhp(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".php")
	m := martini.Classic()
	m.Handlers(fh.ServeHTTP, ph.ServeHTTP)

	w := get(t, m, "/plain.txt")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "This is not PHP\n")

	w = get(t, m, "/abc.php")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc")
}

func TestNoDirectoryListing(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".php")
	m := martini.Classic()
	m.Handlers(fh.ServeHTTP, ph.ServeHTTP)

	w := get(t, m, "/")

	assert.Equal(t, w.Code, 404)
	assert.Contains(t, w.Body.String(), `The requested resource <code class="url">/</code> was not found on this server.`)
}

func TestNonExistent(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".php")
	m := martini.Classic()
	m.Handlers(fh.ServeHTTP, ph.ServeHTTP)

	w := get(t, m, "/404.notfound")

	assert.Equal(t, w.Code, 404)
	assert.Contains(t, w.Body.String(), `The requested resource <code class="url">/404.notfound</code> was not found on this server.`)
}

func TestWpMultisite(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".php")
	m := martini.Classic()
	m.Handlers(fh.ServeHTTP, ph.ServeHTTP)

	// In WordPress we would see URLs like /sitename/wp-content/themes/xyz/assets/img/123.png
	w := get(t, m, "/x/plain.txt")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "This is not PHP\n")

	// Oops, make sure this passes
	w = get(t, m, "/x/plain.txt")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "This is not PHP\n")
}

func TestPhpWithPath(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{}, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".php")
	m := martini.Classic()
	m.Handlers(fh.ServeHTTP, ph.ServeHTTP)

	w := get(t, m, "/abc.php/123")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc")
}
