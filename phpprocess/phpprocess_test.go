package phpprocess

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// func TestListenOnDifferentPorts(t *testing.T) {
// 	ph1, err := NewPhpProcess("test-dir")
// 	assert.Nil(t, err)
// 	ph2, err := NewPhpProcess("test-dir")
// 	assert.Nil(t, err)

// 	assert.NotEqual(t, ph1.host, ph2.host)
// }

func TestErrorWhenListeningOnSamePort(t *testing.T) {
	p1, err := runPhp("test-dir", "localhost:8000")
	defer p1.Kill()
	assert.Nil(t, err)

	p2, err := runPhp("test-dir", "localhost:8000")
	defer p2.Kill()
	assert.NotNil(t, err)
}
