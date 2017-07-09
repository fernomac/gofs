package gofs

import (
	"errors"
	"io"
	"os"
)

// Mock implementation of the gofs.File interface.
type mockFile struct {
	name     string
	info     *mockFileInfo
	position int
}

func (f *mockFile) Name() string {
	return f.name
}

func (f *mockFile) Stat() (os.FileInfo, error) {
	return f.info, nil
}

func (f *mockFile) Chmod(mode os.FileMode) error {
	f.info.mode = (f.info.mode & os.ModeType) | (mode & os.ModePerm)
	return nil
}

func (f *mockFile) Readdir(n int) ([]os.FileInfo, error) {
	if !f.info.mode.IsDir() {
		return nil, errors.New("not a directory")
	}

	var ret []os.FileInfo
	for _, v := range f.info.children {
		if n > 0 && len(ret) >= n {
			break
		}
		ret = append(ret, v)
	}
	return ret, nil
}

func (f *mockFile) Read(b []byte) (int, error) {
	if f.position == -1 {
		return 0, errors.New("closed")
	}
	if !f.info.mode.IsRegular() {
		return 0, errors.New("not a regular file")
	}
	if f.position >= len(f.info.data) {
		return 0, io.EOF
	}
	ret := copy(b, f.info.data[f.position:])
	f.position += ret
	return ret, nil
}

func (f *mockFile) Write(b []byte) (int, error) {
	if f.position == -1 {
		return 0, errors.New("closed")
	}
	if !f.info.mode.IsRegular() {
		return 0, errors.New("not a regular file")
	}

	pos := 0
	for pos < len(b) {
		l := len(b) - pos
		if f.position == len(f.info.data) {
			// Append.
			f.info.data = append(f.info.data, b[pos:]...)
			f.position += l
			pos += l
		} else {
			// In-place write.
			copied := copy(f.info.data[f.position:], b[pos:])
			f.position += copied
			pos += copied
		}
	}
	return pos, nil
}

func (f *mockFile) Seek(offset int64, whence int) (int64, error) {
	if f.position == -1 {
		return 0, errors.New("closed")
	}
	if !f.info.mode.IsRegular() {
		return 0, errors.New("not a regular file")
	}

	switch whence {
	case os.SEEK_SET:
		if offset < 0 || offset > int64(len(f.info.data)) {
			return 0, errors.New("offset out of bounds")
		}
		f.position = int(offset)
	case os.SEEK_CUR:
		if offset < int64(-f.position) || offset > int64(len(f.info.data)-f.position) {
			return 0, errors.New("offset out of bounds")
		}
		f.position = int(int64(f.position) + offset)
	case os.SEEK_END:
		if offset < 0 || offset > int64(len(f.info.data)) {
			return 0, errors.New("offset out of bounds")
		}
		f.position = len(f.info.data) - int(offset)
	}
	return int64(f.position), nil
}

func (f *mockFile) Truncate(size int64) error {
	if size < 0 {
		return errors.New("size out of bounds")
	}
	if !f.info.mode.IsRegular() {
		return errors.New("not a regular file")
	}
	if size < int64(len(f.info.data)) {
		f.info.data = f.info.data[0:size]
	} else {
		buf := make([]byte, size)
		copy(buf, f.info.data)
		f.info.data = buf
	}
	return nil
}

func (f *mockFile) Sync() error {
	// no-op.
	return nil
}

func (f *mockFile) Close() error {
	f.position = -1
	return nil
}
