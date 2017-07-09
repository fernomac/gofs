package gofs

import (
	"io"
	"io/ioutil"
	"os"
	"sort"
)

// File is like os.File, but an interface.
type File interface {
	io.Reader
	io.Writer
	io.Closer

	Name() string
	Stat() (os.FileInfo, error)

	Chmod(mode os.FileMode) error

	Readdir(n int) ([]os.FileInfo, error)

	Seek(offset int64, whence int) (int64, error)
	Truncate(size int64) error

	Sync() error
}

// FileSystem is like the File related portions of the os package, but an interface.
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)

	Getwd() (string, error)
	Chdir(dir string) error

	Abs(path string) (string, error)

	Chmod(name string, mode os.FileMode) error

	Lstat(name string) (os.FileInfo, error)
	Readlink(name string) (string, error)
	Symlink(oldname, newname string) error

	Mkdir(path string, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error

	Open(name string) (File, error)
	Create(name string) (File, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)

	Truncate(name string, size int64) error
	Remove(name string) error
	RemoveAll(path string) error
	Rename(oldpath, newpath string) error
}

// FileExists checks if a file exists (and is a regular file).
func FileExists(fs FileSystem, path string) (bool, error) {
	info, err := fs.Stat(path)
	if err != nil {
		return false, nil
	}
	return info.Mode().IsRegular(), nil
}

// DirExists checks if a directory exists (and is a directory).
func DirExists(fs FileSystem, path string) (bool, error) {
	info, err := fs.Stat(path)
	if err != nil {
		return false, nil
	}
	return info.Mode().IsDir(), nil
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

// ReadDir is like ioutil.ReadDir, but it takes a FileSystem.
func ReadDir(fs FileSystem, dirname string) ([]os.FileInfo, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}
