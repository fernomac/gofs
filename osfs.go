package gofs

import "os"
import "path/filepath"

type osFilesystem struct {
}

// OsFs creates an OS-based FileSystem.
func OsFs() FileSystem {
	return osFilesystem{}
}

func (osFilesystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (osFilesystem) Getwd() (string, error) {
	return os.Getwd()
}

func (osFilesystem) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (osFilesystem) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (osFilesystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (osFilesystem) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (osFilesystem) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

func (osFilesystem) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (osFilesystem) Open(name string) (File, error) {
	return os.Open(name)
}

func (osFilesystem) Create(name string) (File, error) {
	return os.Create(name)
}

func (osFilesystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}

func (osFilesystem) Mkdir(path string, perm os.FileMode) error {
	return os.Mkdir(path, perm)
}

func (osFilesystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFilesystem) Truncate(name string, size int64) error {
	return os.Truncate(name, size)
}

func (osFilesystem) Remove(name string) error {
	return os.Remove(name)
}

func (osFilesystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (osFilesystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
