package gofs

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func testFileExists(t *testing.T, fs FileSystem, file string, expected bool) {
	t.Run(fmt.Sprintf("FileExists('%v')", file), func(t *testing.T) {
		exists, err := FileExists(fs, file)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if exists != expected {
			t.Fatalf("Expected %v but got %v", expected, exists)
		}
	})
}

func testDirExists(t *testing.T, fs FileSystem, dir string, expected bool) {
	t.Run(fmt.Sprintf("DirExists('%v')", dir), func(t *testing.T) {
		exists, err := DirExists(fs, dir)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if exists != expected {
			t.Fatalf("expected %v but got %v", expected, exists)
		}
	})
}

func TestExists(t *testing.T) {
	fs := MockFs()
	fs.Mkdir("/foo", os.FileMode(0777))
	fs.Mkdir("/foo/bar", os.FileMode(0700))
	f, _ := fs.Create("/foo/bar/hello")
	f.Write([]byte("Hello World"))
	f.Chmod(os.FileMode(0400))
	f.Close()

	testFileExists(t, fs, "/", false)
	testFileExists(t, fs, "/bogus", false)
	testFileExists(t, fs, "/foo", false)
	testFileExists(t, fs, "/foo/bogus", false)
	testFileExists(t, fs, "/foo/bar", false)
	testFileExists(t, fs, "/foo/bar/bogus", false)
	testFileExists(t, fs, "/foo/bar/hello", true)
	testFileExists(t, fs, "/foo/bar/hello/bogus", false)

	testDirExists(t, fs, "/", true)
	testDirExists(t, fs, "/bogus", false)
	testDirExists(t, fs, "/foo", true)
	testDirExists(t, fs, "/foo/bogus", false)
	testDirExists(t, fs, "/foo/bar", true)
	testDirExists(t, fs, "/foo/bar/bogus", false)
	testDirExists(t, fs, "/foo/bar/hello", false)
	testDirExists(t, fs, "/foo/bar/hello/bogus", false)
}

func testOpenFails(t *testing.T, fs FileSystem, path string) {
	t.Run(fmt.Sprintf("Open('%v')", path), func(t *testing.T) {
		_, err := fs.Open(path)
		if err == nil {
			t.Fatalf("Expected an error, got nil")
		}
	})
}

func TestOpen(t *testing.T) {
	fs := MockFs()
	fs.Mkdir("/foo", os.FileMode(0777))
	fs.MkdirAll("/foo/bar/baz", os.FileMode(0700))
	f, _ := fs.Create("/foo/bar/baz/hello")
	f.Write([]byte("Hello World"))
	f.Close()

	testOpenFails(t, fs, "/bogus")
	testOpenFails(t, fs, "/foo/bogus")
	testOpenFails(t, fs, "/foo/bar/bogus")
	testOpenFails(t, fs, "/foo/bar/baz/bogus")

	t.Run("Open('/foo/bar/baz')", func(t *testing.T) {
		f, err := fs.Open("/foo/bar/baz")
		if err != nil {
			t.Fatalf("Unexpected error from Open: %v", err)
		}
		defer f.Close()

		infos, err := f.Readdir(-1)
		if err != nil {
			t.Fatalf("Unexpected error from Readdir: %v", err)
		}

		if len(infos) != 1 {
			t.Fatalf("Unexpected number of files: %v", len(infos))
		}
		if infos[0].Name() != "hello" {
			t.Fatalf("Unexpected name: %v", infos[0].Name())
		}
	})

	t.Run("Open('/foo/bar/baz/hello')", func(t *testing.T) {
		f, err := fs.Open("/foo/bar/baz/hello")
		if err != nil {
			t.Fatalf("Unexpected error from Open: %v", err)
		}
		defer f.Close()

		buf := make([]byte, 8)
		n, err := f.Read(buf)
		if err != nil {
			t.Fatalf("Unexpected error from Read: %v", err)
		}
		if n != 8 {
			t.Fatalf("Unexpected length: expected 8 was %v", n)
		}
		if string(buf) != "Hello Wo" {
			t.Fatalf("Unexpected read result: '%v'", string(buf))
		}

		n, err = f.Read(buf)
		if err != nil {
			t.Fatalf("Unexpected error from Read: %v", err)
		}
		if n != 3 {
			t.Fatalf("Unexpected length: expected 3 was %v", n)
		}
		if string(buf[0:3]) != "rld" {
			t.Fatalf("Unexpected read result: '%v'", string(buf[0:3]))
		}

		_, err = f.Read(buf)
		if err != io.EOF {
			t.Fatalf("Expected EOF, got '%v'", err)
		}
	})
}
