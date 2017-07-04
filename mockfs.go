package gofs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type mockFileSystem struct {
	root mockFileInfo
	cwd  string
}

// MockFs creates a new mock FileSystem
func MockFs() FileSystem {
	return &mockFileSystem{
		root: mockFileInfo{
			name:     "/",
			mode:     os.ModeDir | os.FileMode(0755),
			children: make(map[string]*mockFileInfo),
			data:     nil,
		},
		cwd: "/",
	}
}

func (fs *mockFileSystem) toAbs(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	return filepath.Join(fs.cwd, path)
}

func split(path string) (string, string) {
	return filepath.Dir(path), filepath.Base(path)
}

func (fs *mockFileSystem) toAbsSplit(path string) (string, string) {
	abs := fs.toAbs(path)
	return split(abs)
}

func (fs *mockFileSystem) find(path string, deref bool) (*mockFileInfo, error) {
	if path == "" || path == "/" {
		return &fs.root, nil
	}

	dirPath := filepath.Dir(path)
	dirInfo, err := fs.find(dirPath, true)
	if err != nil {
		return nil, err
	}
	if !dirInfo.mode.IsDir() {
		return nil, fmt.Errorf("'%v' is not a directory", dirPath)
	}

	info := dirInfo.children[filepath.Base(path)]
	if info == nil {
		return nil, fmt.Errorf("'%v' not found", path)
	}

	if deref {
		return fs.deref(info)
	}
	return info, nil
}

func (fs *mockFileSystem) deref(info *mockFileInfo) (*mockFileInfo, error) {
	if info.mode&os.ModeSymlink == 0 {
		return info, nil
	}
	return fs.find(string(info.data), true)
}

func (fs *mockFileSystem) stat(name string) (*mockFileInfo, error) {
	return fs.find(fs.toAbs(name), true)
}

func (fs *mockFileSystem) Stat(name string) (os.FileInfo, error) {
	return fs.stat(name)
}

func (fs *mockFileSystem) Getwd() (string, error) {
	return fs.cwd, nil
}

func (fs *mockFileSystem) Chdir(dir string) error {
	info, err := fs.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("chdir: not a dir")
	}
	fs.cwd = dir
	return nil
}

func (fs *mockFileSystem) Chmod(name string, mode os.FileMode) error {
	info, err := fs.stat(name)
	if err != nil {
		return err
	}
	f := mockFile{info: info}
	return f.Chmod(mode)
}

func (fs *mockFileSystem) lstat(name string) (*mockFileInfo, error) {
	return fs.find(fs.toAbs(name), false)
}

func (fs *mockFileSystem) Lstat(name string) (os.FileInfo, error) {
	return fs.lstat(name)
}

func (fs *mockFileSystem) Readlink(name string) (string, error) {
	info, err := fs.lstat(name)
	if err != nil {
		return "", err
	}
	if info.mode&os.ModeSymlink == 0 {
		return "", errors.New("realink: not a symlink")
	}
	return string(info.data), nil
}

func (fs *mockFileSystem) Symlink(oldname, newname string) error {
	oldAbs := fs.toAbs(oldname)
	newAbs := fs.toAbs(newname)

	dirPath, fileName := split(newAbs)
	dirInfo, err := fs.find(dirPath, true)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return errors.New("symlink: not a directory")
	}

	info := dirInfo.children[fileName]
	if info != nil {
		return errors.New("symlink: already exists")
	}

	info = &mockFileInfo{
		name:     newAbs,
		mode:     os.ModeSymlink | os.FileMode(0777),
		children: nil,
		data:     []byte(oldAbs),
	}

	dirInfo.children[fileName] = info
	return nil
}

func (fs *mockFileSystem) Mkdir(path string, perm os.FileMode) error {
	abs := fs.toAbs(path)
	dirPath, fileName := split(abs)

	dirInfo, err := fs.find(dirPath, true)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("mkdir: parent '%v' exists but is not a directory", dirPath)
	}

	info := dirInfo.children[fileName]
	if info != nil {
		if info.IsDir() {
			// Alread exists.
			return nil
		}
		return fmt.Errorf("mkdir: '%v' already exists but is not a directory", abs)
	}

	info = &mockFileInfo{
		name:     abs,
		mode:     os.ModeDir | (perm & os.ModePerm),
		children: make(map[string]*mockFileInfo),
		data:     nil,
	}
	dirInfo.children[fileName] = info
	return nil
}

func (fs *mockFileSystem) doMkdirAll(path string, perm os.FileMode) (*mockFileInfo, error) {
	if path == "" || path == "/" {
		return &fs.root, nil
	}

	dirPath, fileName := split(path)
	dirInfo, err := fs.doMkdirAll(dirPath, perm)
	if err != nil {
		return nil, err
	}

	info := dirInfo.children[fileName]
	if info != nil {
		if info.IsDir() {
			// Already exists and is a dir.
			return info, nil
		}
		return nil, fmt.Errorf("mkdirall: '%v' exists but is not a directory", path)
	}

	info = &mockFileInfo{
		name:     path,
		mode:     os.ModeDir | (perm & os.ModePerm),
		children: make(map[string]*mockFileInfo),
		data:     nil,
	}
	dirInfo.children[fileName] = info
	return info, nil
}

func (fs *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	_, err := fs.doMkdirAll(fs.toAbs(path), perm)
	return err
}

func (fs *mockFileSystem) Open(name string) (File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *mockFileSystem) Create(name string) (File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0666))
}

func (fs *mockFileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	var info *mockFileInfo
	var err error
	abs := fs.toAbs(name)

	if flag&os.O_CREATE == 0 {
		// The file must already exist.
		info, err = fs.find(abs, true)
		if err != nil {
			return nil, err
		}
	} else {
		// We can create the file if needed.
		dirPath, fileName := split(abs)
		dirInfo, err := fs.find(dirPath, true)
		if err != nil {
			return nil, err
		}
		if !dirInfo.IsDir() {
			return nil, errors.New("openfile: not a directory")
		}

		info = dirInfo.children[fileName]
		if info == nil {
			// Create a new one.
			info = &mockFileInfo{
				name:     abs,
				mode:     (perm & os.ModePerm),
				children: nil,
				data:     nil,
			}
			dirInfo.children[fileName] = info
		} else {
			// It already exists.
			if flag&os.O_EXCL == os.O_EXCL {
				return nil, errors.New("openfile: already exists")
			}
			// Handle symlinks.
			info, err = fs.deref(info)
			if err != nil {
				return nil, err
			}
		}
	}

	if !info.mode.IsRegular() {
		return nil, errors.New("openfile: not a regular file")
	}

	// Handle truncate and append flags.
	if flag&os.O_TRUNC == os.O_TRUNC {
		info.data = nil
	}
	position := 0
	if flag&os.O_APPEND == os.O_APPEND {
		position = len(info.data)
	}

	return &mockFile{
		info:     info,
		position: position,
	}, nil
}

func (fs *mockFileSystem) Truncate(name string, size int64) error {
	info, err := fs.stat(name)
	if err != nil {
		return err
	}
	file := mockFile{info: info}
	return file.Truncate(size)
}

func (fs *mockFileSystem) Remove(name string) error {
	dirPath, fileName := fs.toAbsSplit(name)

	dirInfo, err := fs.find(dirPath, true)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("'%v' is not a directory", dirPath)
	}

	// Explicitly not following symlinks here; we want to delete the link.
	info := dirInfo.children[fileName]
	if info == nil {
		return fmt.Errorf("remove: '%v' does not exist", name)
	}

	if info.IsDir() && len(info.children) != 0 {
		return fmt.Errorf("remove: '%v' is a non-empty directory", name)
	}

	delete(dirInfo.children, fileName)
	return nil
}

func (fs *mockFileSystem) doRemoveAll(dirInfo *mockFileInfo, fileName string) {
	info := dirInfo.children[fileName]
	if info != nil {
		if info.IsDir() {
			for fn := range info.children {
				fs.doRemoveAll(info, fn)
			}
		}
		delete(info.children, fileName)
	}
}

func (fs *mockFileSystem) RemoveAll(path string) error {
	dirPath, fileName := fs.toAbsSplit(path)
	dirInfo, err := fs.find(dirPath, true)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return errors.New("removeall: not a directory")
	}
	fs.doRemoveAll(dirInfo, fileName)
	return nil
}

func (fs *mockFileSystem) Rename(oldpath, newpath string) error {
	oldDirPath, oldFileName := fs.toAbsSplit(oldpath)
	newDirPath, newFileName := fs.toAbsSplit(newpath)

	oldDirInfo, err := fs.find(oldDirPath, true)
	if err != nil {
		return err
	}
	if !oldDirInfo.IsDir() {
		return fmt.Errorf("rename: '%v' is not a directory", oldDirPath)
	}

	newDirInfo := oldDirInfo
	if oldDirPath != newDirPath {
		newDirInfo, err = fs.find(newDirPath, true)
		if err != nil {
			return err
		}
		if !newDirInfo.IsDir() {
			return fmt.Errorf("rename: '%v' is not a directory", newDirPath)
		}
	}

	info := oldDirInfo.children[oldFileName]
	if info == nil {
		return fmt.Errorf("rename: '%v' not found", oldpath)
	}

	delete(oldDirInfo.children, oldFileName)
	newDirInfo.children[newFileName] = info
	return nil
}
