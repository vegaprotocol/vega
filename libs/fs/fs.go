package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrIsADirectory = errors.New("is a directory")

// EnsureDir will make sure a directory exists or is created at the given path.
func EnsureDir(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, os.ModeDir|0700)
		}
		return err
	}
	return nil
}

// PathExists returns whether a link exists at the given path.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// FileExists similar to PathExists, but ensures the path is to a file, not a
// directory.
func FileExists(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err == nil {
		if fileInfo.IsDir() {
			return false, ErrIsADirectory
		}
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ReadFile(path string) ([]byte, error) {
	dir, fileName := filepath.Split(path)
	if len(dir) == 0 {
		dir = "."
	}

	buf, err := fs.ReadFile(os.DirFS(dir), fileName)
	if err != nil {
		return nil, fmt.Errorf("couldn't read file: %w", err)
	}

	return buf, nil
}

func WriteFile(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("couldn't create file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(content)
	if err != nil {
		return fmt.Errorf("couldn't write file: %w", err)
	}

	return nil
}
