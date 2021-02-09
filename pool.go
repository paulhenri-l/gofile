package gofile

import (
	"github.com/paulhenri-l/gofile/contracts"
	"github.com/pkg/errors"
	"sync"
)

type Pool struct {
	managersPool []contracts.FileManager
	size         int
	mtx          *sync.Mutex
	cnd          *sync.Cond
	blocked      bool
	closed       bool
	writtenBytes uint64
}

func NewPool(managers []contracts.FileManager) *Pool {
	mtx := &sync.Mutex{}
	cnd := sync.NewCond(mtx)

	return &Pool{
		managersPool: managers,
		mtx:          mtx,
		cnd:          cnd,
		blocked:      false,
		size:         len(managers),
	}
}

func (p *Pool) Write(b []byte) (int, error) {
	if p.closed {
		return 0, errors.New("pool closed")
	}

	m := p.take()

	written, err := m.Write(b)
	if err != nil {
		return 0, errors.Wrap(err, "manager write error")
	}

	p.writtenBytes = p.writtenBytes + uint64(written)
	p.put(m)

	return written, nil
}

func (p *Pool) WrittenBytes() uint64 {
	return p.writtenBytes
}

func (p *Pool) Close() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.blocked = true
	p.closed = true

	for len(p.managersPool) < p.size {
		p.cnd.Wait()
	}

	for _, m := range p.managersPool {
		err := m.Close()

		if err != nil {
			return errors.Wrap(err, "unable to close manager")
		}
	}

	p.blocked = false
	return nil
}

func (p *Pool) take() contracts.FileManager {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.blocked == true || len(p.managersPool) <= 0 {
		p.cnd.Wait()
	}

	m, managers := p.managersPool[0], p.managersPool[1:]
	p.managersPool = managers

	return m
}

func (p *Pool) put(m contracts.FileManager) {
	p.mtx.Lock()
	p.managersPool = append(p.managersPool, m)
	p.mtx.Unlock()

	p.cnd.Signal()
}
