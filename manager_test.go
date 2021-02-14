package gofile

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFileManager(t *testing.T) {
	fn := newFileName(t)
	m, _ := NewManager(fn)

	assert.NotNil(t, m)
}

func TestFileIsCreated(t *testing.T) {
	fn := newFileName(t)
	_, _ = NewManager(fn)

	_, err := os.Stat(fn)

	assert.NoError(t, err)
}

func TestFileManagerWritesToFile(t *testing.T) {
	fn := newFileName(t)
	b := []byte("Hello")
	m, _ := NewManager(fn)

	w, err1 := m.Write(b)
	m.Close()
	c, err2 := ioutil.ReadFile(fn)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 5, w)
	assert.Equal(t, b, c)
}

func TestBufferIsFlushedOnClose(t *testing.T) {
	fn := newFileName(t)
	m, _ := NewManager(fn)

	m.Write([]byte("0"))
	c1, _ := ioutil.ReadFile(fn)
	m.Close()
	c2, _ := ioutil.ReadFile(fn)

	assert.Len(t, c1, 0)
	assert.Len(t, c2, 1)
}

func TestTotalWrittenBytes(t *testing.T) {
	m, _ := NewManager(newFileName(t))

	m.Write([]byte("0"))
	m.Write([]byte("0"))
	m.Write([]byte("0"))
	m.Write([]byte("0"))
	m.Write([]byte("0"))

	assert.Equal(t, uint64(5), m.WrittenBytes())
}

func TestManagerCanBeClosed(t *testing.T) {
	m, _ := NewManager(newFileName(t))

	w1, err1 := m.Write([]byte("Hello"))
	m.Close()
	w2, err2 := m.Write([]byte("Hello"))

	assert.Equal(t, 5, w1)
	assert.Equal(t, 0, w2)
	assert.NoError(t, err1)
	assert.Error(t, err2)
}

func TestBrokenCloseWithEmptyBuffer(t *testing.T) {
	fn := newFileName(t)
	m, _ := NewManager(fn)
	m.file.Close()

	err := m.Close()

	assert.Error(t, err)
}

func TestBrokenCloseWithFilledBuffer(t *testing.T) {
	fn := newFileName(t)
	m, _ := NewManager(fn)
	m.Write([]byte("hello"))
	m.file.Close()

	err := m.Close()

	assert.Error(t, err)
}

func TestCloseCanBeCalledMultipleTimes(t *testing.T) {
	m, _ := NewManager(newFileName(t))

	err1 := m.Close()
	err2 := m.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func BenchmarkAppend(b *testing.B) {
	fn := newFileName(b)
	m, _ := NewManager(fn)
	bytes := []byte(strings.Repeat("0", 1024))

	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Write(bytes)
	}
}

func newFileName(t testing.TB) string {
	return NewRandFileName(t.TempDir(), "")
}
