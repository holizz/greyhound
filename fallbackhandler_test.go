package greyhound

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStatic(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".phpx", ph)

	w := get(t, fh, "/abc.php")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "<?php echo 'abc' ?>\n")
}

func TestStaticAndPhp(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{})
	defer ph.Close()
	assert.Nil(t, err)

	fh := NewFallbackHandler("test-dir", ".php", ph)

	w := get(t, fh, "/plain.txt")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "This is not PHP\n")

	w = get(t, fh, "/abc.php")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "abc")
}

func TestNoDirectoryListing(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".php", ph)

	w := get(t, fh, "/")

	assert.Equal(t, w.Code, 404)
	assert.Contains(t, w.Body.String(), `The requested resource <code class="url">/</code> was not found on this server.`)
}

func TestNonExistent(t *testing.T) {
	ph, err := NewPhpHandler("test-dir", time.Second, []string{})
	defer ph.Close()
	assert.Nil(t, err)
	fh := NewFallbackHandler("test-dir", ".php", ph)

	w := get(t, fh, "/404.notfound")

	assert.Equal(t, w.Code, 404)
	assert.Contains(t, w.Body.String(), `The requested resource <code class="url">/404.notfound</code> was not found on this server.`)
}
