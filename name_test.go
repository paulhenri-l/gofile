package gofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamePrefix(t *testing.T) {
	tmp := t.TempDir()
	fn := NewRandFileName(tmp, "my_prefix")

	assert.Contains(t, fn, "my_prefix")
}
