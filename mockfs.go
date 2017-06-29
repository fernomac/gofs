package gofs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type mockFileData struct {
	mode     os.FileMode
	children map[string]*mockFileData
	data     []byte
}

type mockFile struct {
	*mockFileData
	position int
}

func (f *mockFile) Read(b []byte) (int, error) {
	if f.mode.IsDir() {
		return 0, errors.New("I'm a directory, WTF")
	}
	if f.position == -1 {
		return 0, errors.New("I'm closed WTF")
	}
	if f.position >= len(f.data) {
		return 0, io.EOF
	}
	ret := copy(b, f.data[f.position:])
	f.position += ret
	return ret, nil
}

func (f *mockFile) Write(b []byte) (int, error) {
	if f.mode.IsDir() {
		return 0, errors.New("I'm a directory, WTF")
	}
	if f.position == -1 {
		return 0, errors.New("I'm closed WTF")
	}
	if f.position != len(f.data) {
		return 0, errors.New("Only appending is supported")
	}
	f.data = append(f.data, b...)
	f.position += len(b)
	return len(b), nil
}

func (f *mockFile) Close() error {
	f.position = -1
	return nil
}

func (f *mockFile) Chmod(mode os.FileMode) error {
	if f.position == -1 {
		return errors.New("I'm closed WTF")
	}
	f.mode = (f.mode & os.ModeType) | (mode & os.ModePerm)
	return nil
}

type mockFileSystem struct {
	root mockFileData
	cwd  string
}

// MockFs creates a new mock FileSystem
func MockFs() FileSystem {
	return &mockFileSystem{
		root: mockFileData{
			mode:     os.ModeDir | os.FileMode(0755),
			children: make(map[string]*mockFileData),
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

func (fs *mockFileSystem) toAbsSplit(path string) (string, string) {
	abs := fs.toAbs(path)
	return filepath.Dir(abs), filepath.Base(abs)
}

func (fs *mockFileSystem) find(path string) (*mockFileData, error) {
	if path == "" || path == "/" {
		return &fs.root, nil
	}

	dirPath := filepath.Dir(path)
	dir, err := fs.find(dirPath)
	if err != nil {
		return nil, err
	}
	if !dir.mode.IsDir() {
		return nil, fmt.Errorf("'%v' is not a directory", dirPath)
	}

	file := dir.children[filepath.Base(path)]
	if file == nil {
		return nil, fmt.Errorf("'%v' not found", path)
	}
	return file, nil
}

func (fs *mockFileSystem) FileExists(path string) (bool, error) {
	f, err := fs.find(fs.toAbs(path))
	if err != nil {
		return false, nil
	}
	return f.mode.IsRegular(), nil
}

func (fs *mockFileSystem) DirExists(path string) (bool, error) {
	f, err := fs.find(fs.toAbs(path))
	if err != nil {
		return false, nil
	}
	return f.mode.IsDir(), nil
}

func (fs *mockFileSystem) Open(name string) (File, error) {
	f, err := fs.find(fs.toAbs(name))
	if err != nil {
		return nil, err
	}
	if !f.mode.IsRegular() {
		return nil, fmt.Errorf("'%v' is not a regular file", name)
	}
	return &mockFile{
		mockFileData: f,
		position:     0,
	}, nil
}

func (fs *mockFileSystem) Create(name string) (File, error) {
	dirPath, fileName := fs.toAbsSplit(name)

	dir, err := fs.find(dirPath)
	if err != nil {
		return nil, err
	}
	if !dir.mode.IsDir() {
		return nil, fmt.Errorf("Parent '%v' exists but is not a directory", dirPath)
	}

	file := dir.children[fileName]
	if file != nil {
		return nil, errors.New("File already exists")
	}

	file = &mockFileData{
		mode:     os.FileMode(0666),
		children: nil,
		data:     nil,
	}
	dir.children[fileName] = file

	return &mockFile{
		mockFileData: file,
		position:     0,
	}, nil
}

func (fs *mockFileSystem) Mkdir(path string, perm os.FileMode) error {
	dirPath, fileName := fs.toAbsSplit(path)

	dir, err := fs.find(dirPath)
	if err != nil {
		return err
	}
	if !dir.mode.IsDir() {
		return fmt.Errorf("Parent '%v' exists but is not a directory", dirPath)
	}

	file := dir.children[fileName]
	if file != nil {
		if file.mode.IsDir() {
			// Alread exists.
			return nil
		}
		return errors.New("Already exists but is not a directory")
	}

	file = &mockFileData{
		mode:     os.ModeDir | (perm & os.ModePerm),
		children: make(map[string]*mockFileData),
		data:     nil,
	}
	dir.children[fileName] = file
	return nil
}

func (fs *mockFileSystem) doMkdirAll(path string, perm os.FileMode) (*mockFileData, error) {
	if path == "" || path == "/" {
		return &fs.root, nil
	}

	dirPath := filepath.Dir(path)
	fileName := filepath.Base(path)

	dir, err := fs.doMkdirAll(dirPath, perm)
	if err != nil {
		return nil, err
	}

	file := dir.children[fileName]
	if file != nil {
		if file.mode.IsDir() {
			// Already exists and is a dir.
			return file, nil
		}
		return nil, fmt.Errorf("'%v' exists but is not a directory", path)
	}

	file = &mockFileData{
		mode:     os.ModeDir | (perm & os.ModePerm),
		children: make(map[string]*mockFileData),
		data:     nil,
	}
	dir.children[fileName] = file
	return file, nil
}

func (fs *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	_, err := fs.doMkdirAll(fs.toAbs(path), perm)
	return err
}

func (fs *mockFileSystem) Remove(name string) error {
	dirPath, fileName := fs.toAbsSplit(name)

	dir, err := fs.find(dirPath)
	if err != nil {
		return err
	}
	if !dir.mode.IsDir() {
		return fmt.Errorf("'%v' is not a directory", dirPath)
	}

	file := dir.children[fileName]
	if file == nil {
		return fmt.Errorf("'%v' does not exist", name)
	}

	if file.mode.IsDir() && len(file.children) != 0 {
		return fmt.Errorf("'%v' is a non-empty directory", name)
	}

	delete(dir.children, fileName)
	return nil
}

func (fs *mockFileSystem) Rename(oldpath, newpath string) error {
	oldDirPath, oldFileName := fs.toAbsSplit(oldpath)
	newDirPath, newFileName := fs.toAbsSplit(newpath)

	oldDir, err := fs.find(oldDirPath)
	if err != nil {
		return err
	}
	if !oldDir.mode.IsDir() {
		return fmt.Errorf("'%v' is not a directory", oldDirPath)
	}

	newDir := oldDir
	if oldDirPath != newDirPath {
		newDir, err := fs.find(newDirPath)
		if err != nil {
			return err
		}
		if !newDir.mode.IsDir() {
			return fmt.Errorf("'%v' is not a directory", newDirPath)
		}
	}

	file := oldDir.children[oldFileName]
	if file == nil {
		return fmt.Errorf("'%v' not found", oldpath)
	}

	delete(oldDir.children, oldFileName)
	newDir.children[newFileName] = file
	return nil
}
