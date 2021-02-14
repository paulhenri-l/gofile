package gofile

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/paulhenri-l/gofile/contracts"
	m "github.com/paulhenri-l/gofile/mocks/contracts"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewRotatingManagerWithFactory_WithBrokenFactory(t *testing.T) {
	_, err := NewRotatingManagerWithFactory(
		t.TempDir(), "test_", time.Second, 1000, newBrokenManagerFactory(),
	)

	assert.Error(t, err)
}

func TestNoRotationWhenNothingWritten(t *testing.T) {
	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Millisecond*1, 1000)

	m1 := rm.m
	time.Sleep(time.Millisecond * 10)
	m2 := rm.m

	assert.Equal(t, m1, m2)
}

func TestRotationEveryNSeconds(t *testing.T) {
	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Millisecond*1, 1000)
	_, _ = rm.Write([]byte("hello"))

	rm.mtx.Lock()
	m1 := rm.m
	rm.mtx.Unlock()

	time.Sleep(time.Millisecond * 10)

	rm.mtx.Lock()
	m2 := rm.m
	rm.mtx.Unlock()

	assert.NotEqual(t, m1, m2)
}

func TestRotationEveryNBytes(t *testing.T) {
	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Second*100, 5)
	m1 := rm.m

	_, _ = rm.Write([]byte("hello"))
	m2 := rm.m

	assert.NotEqual(t, m1, m2)
}

func TestRotationEveryNBytesRefreshedWhenRotated(t *testing.T) {
	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Second*100, 5)

	_, _ = rm.Write([]byte("hello"))
	_, _ = rm.Write([]byte("hello"))
	_, _ = rm.Write([]byte("hello"))
	_, _ = rm.Write([]byte("hello"))
	_, _ = rm.Write([]byte("hello"))
	m1 := rm.m
	_, _ = rm.Write([]byte("hi"))
	m2 := rm.m

	assert.Equal(t, m1, m2)
}

func TestHandlerCalledOnEveryRotation(t *testing.T) {
	var called bool
	var receivedPath string
	cb := func(path string) {
		receivedPath = path
		called = true
	}

	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Second*100, 5)
	rm.WithRotatedFileHandler(cb)

	_, _ = rm.Write([]byte("hello"))

	assert.True(t, called)
	assert.NotEqual(t, "", receivedPath)
}

func TestWritesAreForwarded(t *testing.T) {
	b := []byte("hello")
	f, _, m := newTestManagerFactory(t)

	m.EXPECT().Write(gomock.Eq(b)).Return(5, nil)
	m.EXPECT().WrittenBytes().Return(uint64(5))

	rm, _ := NewRotatingManagerWithFactory(
		t.TempDir(), "events_", time.Second*100, 1000, f,
	)

	_, _ = rm.Write(b)
}

func TestConcurrentWrites(t *testing.T) {
	var b []byte
	tmp := t.TempDir()
	rm, _ := NewRotatingManager(
		tmp, "events_", time.Millisecond*5, 10,
	)

	wg := sync.WaitGroup{}
	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func(i int) {
			rm.Write([]byte(fmt.Sprintf("Hello_%d\n", i)))
			wg.Done()
		}(i)
	}
	wg.Wait()

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

func TestClosedWhenRotating(t *testing.T) {
	b := []byte("helloo")
	f, _, m := newTestManagerFactory(t)

	m.EXPECT().Write(gomock.Eq(b)).Return(5, nil)
	m.EXPECT().WrittenBytes().Return(uint64(5))
	m.EXPECT().Close()

	rm, _ := NewRotatingManagerWithFactory(
		t.TempDir(), "events_", time.Second*100, 5, f,
	)

	rm.Write(b)
}

func TestWrittenBytesAreKeptBetweenRotations(t *testing.T) {
	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Second*100, 5)

	rm.Write([]byte("hello"))
	rm.Write([]byte("hello"))
	time.Sleep(time.Millisecond * 10)

	assert.Equal(t, uint64(10), rm.WrittenBytes())
}

func TestRotatingManager_RotationBrokenClose(t *testing.T) {
	assert.Panics(t, func() {
		b := []byte("hello")
		f, _, m := newTestManagerFactory(t)

		m.EXPECT().Write(gomock.Eq(b)).Return(5, nil)
		m.EXPECT().WrittenBytes().Return(uint64(5))
		m.EXPECT().Close().Return(errors.New("I am broken"))

		rm, _ := NewRotatingManagerWithFactory(
			t.TempDir(), "events_", time.Second*100, 5, f,
		)

		rm.Write(b)
	})
}

func TestRotatingManager_RotationBrokenFactory(t *testing.T) {
	assert.Panics(t, func() {
		b := []byte("hello")
		f, _, m := newTestManagerFactory(t)

		m.EXPECT().Write(gomock.Eq(b)).Return(5, nil)
		m.EXPECT().WrittenBytes().Return(uint64(5))
		m.EXPECT().Close()

		rm, _ := NewRotatingManagerWithFactory(
			t.TempDir(), "events_", time.Second*100, 5, f,
		)

		rm.factory = newBrokenManagerFactory()

		rm.Write(b)
	})
}

func TestCannotWriteOnStoppedRotatingManager(t *testing.T) {
	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Second*100, 5)

	rm.Close()
	w, err := rm.Write([]byte("hello"))

	assert.Equal(t, 0, w)
	assert.Error(t, err)
}

func TestClose(t *testing.T) {
	f, _, m := newTestManagerFactory(t)

	m.EXPECT().Close()

	rm, _ := NewRotatingManagerWithFactory(
		t.TempDir(), "events_", time.Second*100, 5, f,
	)

	rm.Close()
}

func TestHandlerCalledOnClose(t *testing.T) {
	var called bool
	var receivedPath string
	cb := func(path string) {
		receivedPath = path
		called = true
	}

	rm, _ := NewRotatingManager(t.TempDir(), "events_", time.Second*100, 5)
	rm.WithRotatedFileHandler(cb)

	_, _ = rm.Write([]byte("1"))
	rm.Close()

	assert.True(t, called)
	assert.NotEqual(t, "", receivedPath)
}

func TestRotatingManager_Stop_CloseError(t *testing.T) {
	f, _, m := newTestManagerFactory(t)

	m.EXPECT().Close().Return(errors.New("I am broken"))

	rm, _ := NewRotatingManagerWithFactory(
		t.TempDir(), "events_", time.Second*100, 5, f,
	)

	err := rm.Close()

	assert.Error(t, err)
}

func BenchmarkRotatingManager_Write(b *testing.B) {
	tmp := fakeTmpPath(b)
	rm, _ := NewRotatingManager(
		tmp, "test_", time.Second * 1, 1024 * 1024 * 100,
	)

	bytes := []byte(strings.Repeat("0", 1024))

	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.Write(bytes)
	}
	rm.Close()
}

func newTestManagerFactory(t *testing.T) (ManagerFactory, *gomock.Controller, *m.MockFileManager) {
	ctl := gomock.NewController(t)
	m := m.NewMockFileManager(ctl)
	t.Cleanup(func() {
		ctl.Finish()
	})

	f := func(_ string) (contracts.FileManager, error) {
		return m, nil
	}

	return f, ctl, m
}

func newBrokenManagerFactory() ManagerFactory {
	return func(fileName string) (contracts.FileManager, error) {
		return nil, errors.New("I am broken")
	}
}
