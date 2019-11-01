package storage

import (
	"fmt"
	"io/ioutil"
	"os"
)

func noop() {
	// NO-OP
}

// TempDir will create a temporary folderwith the give prefix
func TempDir(prefix string) (string, func(), error) {
	baseTempDirs := []string{"/dev/shm", os.TempDir()}
	for _, baseTempDir := range baseTempDirs {
		_, err := os.Stat(baseTempDir)
		if err == nil {
			dir, err := ioutil.TempDir(baseTempDir, prefix)
			if err != nil {
				return "", noop, fmt.Errorf("Could not create tmp dir in %s", baseTempDir)
			}
			return dir, func() { os.RemoveAll(dir) }, nil
		}
	}
	return "", noop, fmt.Errorf("Could not find a temp dir")
}
