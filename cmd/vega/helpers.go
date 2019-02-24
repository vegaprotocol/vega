package main

import (
	"fmt"
	"os"
)

const (
	dirPerms = 0700
)

func DefaultVegaDir() string {
	return os.ExpandEnv("$HOME/.vega")
}

func EnsureDir(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Mkdir(path, dirPerms)
		}
		return err
	}
	return nil
}

type NotFound struct {
	path string
}

func (err *NotFound) Error() string {
	return fmt.Sprintf("not found: %s", err.path)
}

// Exists returns whether a link exists at a given filesystem path.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, &NotFound{path}
	}
	return false, err
}

//func CreateUnlessExists(path string) error {
//	if exists, _ := Exists(path); exists {
//		return fmt.Errorf("directory `%v` already exists", path)
//	}
//	if err := os.Mkdir(path, dirPerms); err != nil {
//		return fmt.Errorf("could not create directory `%v` (%v)", path, err)
//	}
//	return nil
//}
