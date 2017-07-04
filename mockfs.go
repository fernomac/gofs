package gofs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DebugFs is a file system that can dump its state for debug purposes.
type DebugFs interface {
	Dump()
}

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
			parent:   nil,
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
	if !strings.HasPrefix(path, "/") {
		return nil, errors.New("path is not absolute")
	}

	dirPath := filepath.Dir(path)
	dirInfo, err := fs.find(dirPath, true)
	if err != nil {
		return nil, err
	}
	if !dirInfo.mode.IsDir() {
		return nil, os.ErrNotExist
	}

	info := dirInfo.children[filepath.Base(path)]
	if info == nil {
		return nil, os.ErrNotExist
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

func (fs *mockFileSystem) findDir(op string, path string) (*mockFileInfo, error) {
	info, err := fs.find(path, true)
	if err != nil {
		return nil, &os.PathError{
			Op:   op,
			Err:  err,
			Path: path,
		}
	}
	if !info.IsDir() {
		return nil, &os.PathError{
			Op:   op,
			Err:  errors.New("not a directory"),
			Path: path,
		}
	}
	return info, nil
}

func (fs *mockFileSystem) stat(name string) (*mockFileInfo, error) {
	info, err := fs.find(fs.toAbs(name), true)
	if err != nil {
		return nil, &os.PathError{
			Op:   "stat",
			Err:  err,
			Path: name,
		}
	}
	return info, nil
}

func (fs *mockFileSystem) Stat(name string) (os.FileInfo, error) {
	return fs.stat(name)
}

func (fs *mockFileSystem) Getwd() (string, error) {
	return fs.cwd, nil
}

func (fs *mockFileSystem) Chdir(dir string) error {
	abs := fs.toAbs(dir)
	_, err := fs.findDir("chdir", abs)
	if err == nil {
		fs.cwd = abs
	}
	return err
}

func (fs *mockFileSystem) Abs(path string) (string, error) {
	return fs.toAbs(path), nil
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
	info, err := fs.find(fs.toAbs(name), false)
	if err != nil {
		return nil, &os.PathError{
			Op:   "lstat",
			Err:  err,
			Path: name,
		}
	}
	return info, nil
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
		return "", &os.PathError{
			Op:   "readlink",
			Err:  errors.New("not a symlink"),
			Path: name,
		}
	}
	return string(info.data), nil
}

func (fs *mockFileSystem) Symlink(oldname, newname string) error {
	oldAbs := fs.toAbs(oldname)
	newAbs := fs.toAbs(newname)

	dirPath, fileName := split(newAbs)
	dirInfo, err := fs.findDir("symlink", dirPath)
	if err != nil {
		return err
	}

	info := dirInfo.children[fileName]
	if info != nil {
		return &os.PathError{
			Op:   "symlink",
			Err:  os.ErrExist,
			Path: newname,
		}
	}

	info = &mockFileInfo{
		name:     fileName,
		mode:     os.ModeSymlink | os.FileMode(0777),
		parent:   dirInfo,
		children: nil,
		data:     []byte(oldAbs),
	}

	dirInfo.children[fileName] = info
	return nil
}

func (fs *mockFileSystem) Mkdir(path string, perm os.FileMode) error {
	abs := fs.toAbs(path)
	dirPath, fileName := split(abs)

	dirInfo, err := fs.findDir("mkdir", dirPath)
	if err != nil {
		return err
	}

	info := dirInfo.children[fileName]
	if info != nil {
		if info.IsDir() {
			// Already exists.
			return nil
		}
		return &os.PathError{
			Op:   "mkdir",
			Err:  errors.New("exists but not a directory"),
			Path: abs,
		}
	}

	info = &mockFileInfo{
		name:     fileName,
		mode:     os.ModeDir | (perm & os.ModePerm),
		parent:   dirInfo,
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
		return nil, &os.PathError{
			Op:   "mkdirall",
			Err:  errors.New("exists but not a directory"),
			Path: path,
		}
	}

	info = &mockFileInfo{
		name:     fileName,
		mode:     os.ModeDir | (perm & os.ModePerm),
		parent:   dirInfo,
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
			return nil, &os.PathError{
				Op:   "openfile",
				Err:  err,
				Path: name,
			}
		}
	} else {
		// We can create the file if needed.
		dirPath, fileName := split(abs)
		dirInfo, err := fs.findDir("openfile", dirPath)
		if err != nil {
			return nil, err
		}

		info = dirInfo.children[fileName]
		if info == nil {
			// Create a new one.
			info = &mockFileInfo{
				name:     fileName,
				mode:     (perm & os.ModePerm),
				parent:   dirInfo,
				children: nil,
				data:     nil,
			}
			dirInfo.children[fileName] = info
		} else {
			// It already exists.
			if flag&os.O_EXCL == os.O_EXCL {
				return nil, &os.PathError{
					Op:   "openfile",
					Err:  os.ErrExist,
					Path: name,
				}
			}
			// Handle symlinks.
			info, err = fs.deref(info)
			if err != nil {
				return nil, &os.PathError{
					Op:   "openfile",
					Err:  err,
					Path: name,
				}
			}
		}
	}

	if !info.mode.IsRegular() {
		return nil, &os.PathError{
			Op:   "openfile",
			Err:  errors.New("not a regular file"),
			Path: name,
		}
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

	dirInfo, err := fs.findDir("remove", dirPath)
	if err != nil {
		return err
	}

	// Explicitly not following symlinks here; we want to delete the link.
	info := dirInfo.children[fileName]
	if info == nil {
		return &os.PathError{
			Op:   "remove",
			Err:  os.ErrNotExist,
			Path: name,
		}
	}

	if info.IsDir() && len(info.children) != 0 {
		return &os.PathError{
			Op:   "remove",
			Err:  errors.New("directory is not empty"),
			Path: name,
		}
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
	if err == nil && dirInfo.IsDir() {
		fs.doRemoveAll(dirInfo, fileName)
	}
	return nil
}

func (fs *mockFileSystem) Rename(oldpath, newpath string) error {
	oldDirPath, oldFileName := fs.toAbsSplit(oldpath)
	newDirPath, newFileName := fs.toAbsSplit(newpath)

	oldDirInfo, err := fs.findDir("rename", oldDirPath)
	if err != nil {
		return err
	}

	info := oldDirInfo.children[oldFileName]
	if info == nil {
		return &os.PathError{
			Op:   "rename",
			Err:  os.ErrNotExist,
			Path: oldpath,
		}
	}

	newDirInfo := oldDirInfo
	if oldDirPath != newDirPath {
		newDirInfo, err = fs.findDir("rename", newDirPath)
		if err != nil {
			return err
		}
	}

	delete(oldDirInfo.children, oldFileName)
	info.name = newFileName
	info.parent = newDirInfo
	newDirInfo.children[newFileName] = info
	return nil
}

func dump(info *mockFileInfo) {
	fmt.Println(info.Name())
	if info.IsDir() {
		for _, child := range info.children {
			dump(child)
		}
	}
}

func (fs *mockFileSystem) Dump() {
	dump(&fs.root)
}
