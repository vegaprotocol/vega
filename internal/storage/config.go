package storage

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

const namedLogger = "storage"

func InitStoreDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not create directory path for badger data store: %s", path))
		}
	}
	return nil
}
