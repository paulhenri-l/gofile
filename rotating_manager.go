package file

import (
	"context"
	"github.com/pkg/errors"
	"github.com/paulhenri-l/gofile/chans"
	"github.com/paulhenri-l/gofile/contracts"
	"sync"
	"time"
)

type RotatedFileHandler func(path string)

type decoratedManager struct {
	contracts.FileManager
	path string
}

type RotatingManager struct {
	m                  *decoratedManager
	mtx                *sync.Mutex
	path               string
	prefix             string
	factory            managerFactory
	rotateTime         time.Duration
	rotateTicker       *time.Ticker
	rotateSize         uint64
	rotatedFileHandler RotatedFileHandler
	stopped            bool
	done               chan bool
	ctx                context.Context
	cancel             context.CancelFunc
	writtenBytes       uint64
}

func NewRotatingManager(
	path,
	prefix string,
	rotateTime time.Duration,
	rotateSize uint64,
) (*RotatingManager, error) {
	f := func(path string) (contracts.FileManager, error) {
		return NewManager(path)
	}

	return NewRotatingManagerWithFactory(
		path, prefix, rotateTime, rotateSize, f,
	)
}

func NewRotatingManagerWithFactory(
	path,
	prefix string,
	rotateTime time.Duration,
	rotateSize uint64,
	f managerFactory,
) (*RotatingManager, error) {
	m, err := newDecoratedManager(path, prefix, f)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create new manager")
	}

	rm := &RotatingManager{
		m:          m,
		prefix:     prefix,
		path:       path,
		mtx:        &sync.Mutex{},
		factory:    f,
		rotateTime: rotateTime,
		rotateSize: rotateSize,
		stopped:    false,
		done:       make(chan bool),
	}

	rm.start()

	return rm, nil
}

func (rm *RotatingManager) WithRotatedFileHandler(h RotatedFileHandler) {
	rm.rotatedFileHandler = h
}

func (rm *RotatingManager) Write(b []byte) (int, error) {
	if rm.stopped {
		return 0, errors.New("rotating manager stopped")
	}

	rm.mtx.Lock()
	defer rm.mtx.Unlock()

	w, err := rm.m.Write(b)
	if err != nil {
		return w, errors.Wrap(err, "unable to write to manager")
	}

	rm.writtenBytes = rm.writtenBytes + uint64(w)

	if rm.m.WrittenBytes() >= rm.rotateSize {
		rm.rotate()
	}

	return w, nil
}

func (rm *RotatingManager) WrittenBytes() uint64 {
	return rm.writtenBytes
}

func (rm *RotatingManager) Close() error {
	rm.cancel()
	<-rm.done
	rm.stopped = true

	if err := rm.m.Close(); err != nil {
		return errors.Wrap(err, "unable to close manager")
	}

	rm.notifyRotationHandler()
	return nil
}

func (rm *RotatingManager) start() {
	rm.mtx.Lock()
	defer rm.mtx.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(rm.rotateTime)

	go func() {
		defer ticker.Stop()

		for range chans.OrDoneTimeTime(ctx, ticker.C) {
			rm.mtx.Lock()
			if rm.m.WrittenBytes() > 0 {
				rm.rotate()
			}
			rm.mtx.Unlock()
		}

		rm.done <- true
	}()

	rm.ctx = ctx
	rm.cancel = cancel
	rm.rotateTicker = ticker
}

func (rm *RotatingManager) rotate() {
	if err := rm.m.Close(); err != nil {
		panic(err)
	}

	m, err := newDecoratedManager(rm.path, rm.prefix, rm.factory)
	if err != nil {
		panic(err)
	}

	rm.notifyRotationHandler()

	rm.m = m
	rm.rotateTicker.Reset(rm.rotateTime)
}

func (rm *RotatingManager) notifyRotationHandler() {
	if rm.rotatedFileHandler != nil {
		rm.rotatedFileHandler(rm.m.path)
	}
}

func newDecoratedManager(path, prefix string, f managerFactory) (*decoratedManager, error) {
	fn := newRandFileName(path, prefix)

	m, err := f(fn)
	if err != nil {
		return nil, errors.Wrap(err, "manager factory failed")
	}

	return &decoratedManager{
		FileManager: m,
		path:        fn,
	}, nil
}
