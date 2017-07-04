package gofs

import (
	"os"
	"path/filepath"
	"time"
)

// A mock implementation of the os.FileInfo interface.

type mockFileInfo struct {
	name     string
	mode     os.FileMode
	parent   *mockFileInfo
	children map[string]*mockFileInfo
	data     []byte
}

func (fi *mockFileInfo) Name() string {
	if fi.parent == nil {
		return fi.name
	}
	return filepath.Join(fi.parent.Name(), fi.name)
}

func (fi *mockFileInfo) Size() int64 {
	return int64(len(fi.data))
}

func (fi *mockFileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *mockFileInfo) ModTime() time.Time {
	// Mod times are not supported.
	return time.Unix(0, 0)
}

func (fi *mockFileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (fi *mockFileInfo) Sys() interface{} {
	return nil
}
