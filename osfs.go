package gofs

import "os"

type osFilesystem struct {
}

// OsFs creates an OS-based FileSystem.
func OsFs() FileSystem {
	return osFilesystem{}
}

func (osFilesystem) Getwd() (string, error) {
	return os.Getwd()
}

func (osFilesystem) FileExists(file string) (bool, error) {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !info.IsDir(), nil
}

func (osFilesystem) DirExists(file string) (bool, error) {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
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

func (osFilesystem) Remove(name string) error {
	return os.Remove(name)
}

func (osFilesystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
