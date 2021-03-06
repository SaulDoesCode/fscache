package fscache

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/cornelk/hashmap"
	"gopkg.in/djherbis/stream.v1"
)

type memFS struct {
	files *hashmap.HashMap
}

// NewMemFs creates an in-memory FileSystem.
// It does not support persistence (Reload is a nop).
func NewMemFs() FileSystem {
	return &memFS{
		files: &hashmap.HashMap{},
	}
}

func (fs *memFS) file(name string) (*memFile, bool) {
	raw, ok := fs.files.GetStringKey(name)
	if !ok {
		return nil, ok
	}
	return raw.(*memFile), ok
}

func (fs *memFS) setfile(name string, mf *memFile) {
	if mf != nil {
		fs.files.Set(name, mf)
	}
}

func (fs *memFS) Reload(add func(key, name string)) error {
	return nil
}

func (fs *memFS) AccessTimes(name string) (rt, wt time.Time, err error) {
	f, ok := fs.file(name)
	if ok {
		return f.rt, f.wt, nil
	}
	return rt, wt, errors.New("file has not been read")
}

func (fs *memFS) Create(key string) (stream.File, error) {
	if _, ok := fs.file(key); ok {
		return nil, errors.New("file exists")
	}
	file := &memFile{
		name: key,
		r:    bytes.NewBuffer(nil),
		wt:   time.Now(),
	}
	file.memReader.memFile = file
	fs.setfile(key, file)
	return file, nil
}

func (fs *memFS) Open(name string) (stream.File, error) {
	if f, ok := fs.file(name); ok {
		f.rt = time.Now()
		return &memReader{memFile: f}, nil
	}
	return nil, errors.New("file does not exist")
}

func (fs *memFS) Remove(key string) error {
	fs.files.Del(key)
	return nil
}

func (fs *memFS) RemoveAll() error {
	fs.files = &hashmap.HashMap{}
	return nil
}

type memFile struct {
	mu   sync.RWMutex
	name string
	r    *bytes.Buffer
	memReader
	rt, wt time.Time
}

func (f *memFile) Name() string {
	return f.name
}

func (f *memFile) Write(p []byte) (int, error) {
	if len(p) > 0 {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.r.Write(p)
	}
	return len(p), nil
}

func (f *memFile) Bytes() []byte {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.r.Bytes()
}

func (f *memFile) Close() error {
	return nil
}

type memReader struct {
	*memFile
	n int
}

func (r *memReader) ReadAt(p []byte, off int64) (n int, err error) {
	data := r.Bytes()
	if int64(len(data)) < off {
		return 0, io.EOF
	}
	n, err = bytes.NewReader(data[off:]).ReadAt(p, 0)
	return n, err
}

func (r *memReader) Read(p []byte) (n int, err error) {
	n, err = bytes.NewReader(r.Bytes()[r.n:]).Read(p)
	r.n += n
	return n, err
}

func (r *memReader) Close() error {
	return nil
}
