package gofs

import (
	"io"
	"io/ioutil"
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
	Getwd() (string, error)
	FileExists(file string) (bool, error)
	DirExists(file string) (bool, error)
	Open(name string) (File, error)
	Create(name string) (File, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	Mkdir(path string, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Remove(name string) error
	Rename(oldpath, newpath string) error
}

// ReadFile is like ioutil.ReadFile, but it takes a FileSystem.
func ReadFile(fs FileSystem, filename string) ([]byte, error) {
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// TODO: Optimize based on file size ala the real one.
	return ioutil.ReadAll(file)
}

// WriteFile is like ioutil.WriteFile, but it takes a FileSystem.
func WriteFile(fs FileSystem, filename string, data []byte, perm os.FileMode) error {
	file, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := file.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := file.Close(); err == nil {
		err = err1
	}
	return err
}
