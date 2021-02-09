package file

import (
	"bufio"
	"github.com/pkg/errors"
	"os"
	"sync/atomic"
)

// Manager manages a file for writing, Manager is not threadsafe
// If you need it to be threadsafe you should use the pool instead
type Manager struct {
	path    string
	file    *os.File
	writer  *bufio.Writer
	written uint64
	closed  bool
	deleted bool
}

func NewManager(path string) (*Manager, error) {
	f, err := os.Create(path)

	if err != nil {
		return nil, errors.Wrap(err, "unable to create file")
	}

	return &Manager{
		path:    path,
		file:    f,
		writer:  bufio.NewWriter(f),
		written: 0,
		closed:  false,
		deleted: false,
	}, nil
}

func (m *Manager) Write(b []byte) (int, error) {
	if m.closed != false || m.deleted != false {
		return 0, errors.New("manager closed")
	}

	w, err := m.writer.Write(b)
	if err != nil {
		return 0, errors.Wrap(err, "unable to write to writer")
	}

	atomic.AddUint64(&m.written, uint64(w))

	return w, nil
}

func (m *Manager) WrittenBytes() uint64 {
	return m.written
}

func (m *Manager) Close() error {
	var err error
	err = m.writer.Flush()

	if err != nil {
		return errors.Wrap(err, "unable to flush writer")
	}

	if m.closed != true {
		err = m.file.Close()
		m.closed = true

		if err != nil {
			return errors.Wrap(err, "unable to close managed file")
		}
	}

	return nil
}
