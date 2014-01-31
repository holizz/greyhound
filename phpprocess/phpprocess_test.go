package phpprocess

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestListenOnDifferentPorts(t *testing.T) {
	ph1, err := NewPhpProcess("test-dir")
	defer ph1.Close()
	assert.Nil(t, err)

	ph2, err := NewPhpProcess("test-dir")
	defer ph2.Close()
	assert.Nil(t, err)

	assert.NotEqual(t, ph1.host, ph2.host)
}

func TestRunPhpReturnsErrors(t *testing.T) {
	p1, err := runPhp("test-dir", "localhost:31524")
	defer p1.Kill()
	assert.Nil(t, err)

	p2, err := runPhp("test-dir", "localhost:31524")
	defer p2.Kill()
	assert.NotNil(t, err)
}
