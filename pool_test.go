package gofile

import (
	"fmt"
	"github.com/paulhenri-l/gofile/contracts"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFilePoolCanBeCreated(t *testing.T) {
	m, _ := newFakeManagers(t, 1)
	p := NewPool(m)
	assert.NotNil(t, p)
}

func TestPoolCanTakeWritesConcurrently(t *testing.T) {
	var b []byte
	m, tmp := newFakeManagers(t, 2)
	p := NewPool(m)

	wg := sync.WaitGroup{}
	wg.Add(300)
	for i := 0; i < 300; i++ {
		go func(i int) {
			_, _ = p.Write([]byte(fmt.Sprintf("Hello_%d", i)))
			wg.Done()
		}(i)
	}
	wg.Wait()
	_ = p.Close()

	d, _ := os.Open(tmp)
	files, _ := d.Readdir(0)

	for _, f := range files {
		fb, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", tmp, f.Name()))
		b = append(b, fb...)
	}

	for i := 0; i < 300; i++ {
		assert.Contains(t, string(b), fmt.Sprintf("Hello_%d", i))
	}
}

func TestPoolWriteWait(t *testing.T) {
	m, tmp := newFakeManagers(t, 2)
	p := NewPool(m)
	wait := make(chan struct{})
	_, _ = p.Write([]byte("Hello"))

	go func() {
		m1 := p.take()
		m2 := p.take()
		wait <- struct{}{}
		time.Sleep(200 * time.Millisecond)
		p.put(m1)
		p.put(m2)
	}()

	// Wait for pool to be empty then attempt to write
	// this will block until all managers are back in the pool
	<-wait
	_, _ = p.Write([]byte("Hello"))
	_ = p.Close()

	// Check content
	var c []byte
	d, _ := os.Open(tmp)
	files, _ := d.Readdir(0)

	for _, f := range files {
		fc, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", tmp, f.Name()))
		c = append(c, fc...)
	}

	assert.Contains(t, string(c), "Hello")
}

func TestPoolWrittenBytes(t *testing.T) {
	m, _ := newFakeManagers(t, 2)
	p := NewPool(m)

	w1, _ := p.Write([]byte("0"))
	w2, _ := p.Write([]byte("0"))

	assert.Equal(t, uint64(2), p.WrittenBytes())
	assert.Equal(t, 1, w1)
	assert.Equal(t, 1, w2)
}

func TestFlushing(t *testing.T) {
	m, tmp := newFakeManagers(t, 2)
	p := NewPool(m)
	wait := make(chan struct{})
	_, _ = p.Write([]byte("Hello"))

	go func() {
		m1 := p.take()
		m2 := p.take()
		wait <- struct{}{}
		time.Sleep(200 * time.Millisecond)
		p.put(m1)
		p.put(m2)
	}()

	// Wait for pool to be empty then attempt to flush
	// this will block until all managers are back in the pool
	<-wait
	_ = p.Close()

	// Check flushed content
	var c []byte
	d, _ := os.Open(tmp)
	files, _ := d.Readdir(0)

	for _, f := range files {
		fc, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", tmp, f.Name()))
		c = append(c, fc...)
	}

	assert.Contains(t, string(c), "Hello")
}

func TestWriteAfterClose(t *testing.T) {
	m, _ := newFakeManagers(t, 2)
	p := NewPool(m)

	_ = p.Close()
	_, err := p.Write([]byte("hello"))

	assert.EqualError(t, err, "pool closed")
}

func TestBrokenWrite(t *testing.T) {
	m, _ := newFakeManagers(t, 2)
	p := NewPool(m)

	_ = p.managersPool[0].Close()
	_, err := p.Write([]byte("hello"))

	assert.Error(t, err)
}

func TestBrokenFlush(t *testing.T) {
	m, _ := newFakeManagers(t, 2)
	p := NewPool(m)
	_, _ = p.Write([]byte("hello"))

	p.managersPool[0].(*Manager).file.Close()
	p.managersPool[1].(*Manager).file.Close()

	err := p.Close()

	assert.Error(t, err)
}

func BenchmarkWrites(b *testing.B) {
	bytes := []byte(strings.Repeat("0", 1024))

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("%d file", i)

		b.Run(name, func(b *testing.B) {
			m, _ := newFakeManagers(b, 2)
			p := NewPool(m)

			b.SetBytes(1024)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = p.Write(bytes)
				}
			})
			_ = p.Close()
		})
	}
}

func newFakeManagers(t testing.TB, size int) ([]contracts.FileManager, string) {
	managers := make([]contracts.FileManager, size)
	path := t.TempDir()

	for i := 0; i < size; i++ {
		m, err := NewManager(newRandFileName(path, "test_"))
		if err != nil {
			t.Error(err)
		}

		managers[i] = m
	}

	return managers, path
}