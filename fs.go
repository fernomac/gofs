package gofs

import (
	"io"
	"os"
)

// File is like os.File, but an interface.
type File interface {
	io.Reader
	io.Writer
	io.Closer
	Chmod(mode os.FileMode) error
}

// FileSystem is like the File related portions of the os package, but an interface.
type FileSystem interface {
	FileExists(file string) (bool, error)
	DirExists(file string) (bool, error)
	Open(name string) (File, error)
	Create(name string) (File, error)
	Mkdir(path string, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Remove(name string) error
	Rename(oldpath, newpath string) error
}
