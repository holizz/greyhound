package greyhound

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRunPhpReturnsErrors(t *testing.T) {
	p1, _, _, _, err := runPhp("test-dir", "localhost:31524")
	defer p1.Process.Kill()
	assert.Nil(t, err)

	p2, _, _, _, err := runPhp("test-dir", "localhost:31524")
	defer p2.Process.Kill()
	assert.NotNil(t, err)
}
