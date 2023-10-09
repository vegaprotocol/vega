// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package fsutil

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
)

// RemoveAllFromDirectoryIfExists used in place of os.RemoveAll when the directory should be emptied but not removed.
func RemoveAllFromDirectoryIfExists(dir string) error {
	exists, err := vgfs.PathExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	err = filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		if file != dir {
			err := os.RemoveAll(file)
			if err != nil {
				return fmt.Errorf("failed to remove file:%w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory:%w", err)
	}

	return nil
}

func Md5Hash(path string) (string, error) {
	hash := md5.New()
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type readerAtWrapper struct {
	r io.ReadSeeker
}

func (rw *readerAtWrapper) ReadAt(p []byte, off int64) (n int, err error) {
	// Seek to the requested offset using the underlying io.Reader and io.Seeker
	_, err = rw.r.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return rw.r.Read(p)
}

// ReadNetworkHistorySegmentData takes a io.Reader reading from a network history segment .tar archive then
//   - looks inside the .tar archive for historysnapshot.tar.gz
//   - looks looks historysnapshot.tar.gz for a file called `historyFileName`
//   - returns an io.Reader for reading that file
func ReadNetworkHistorySegmentData(file io.ReadSeekCloser, size int64, historyFileName string) (io.Reader, error) {
	w := &readerAtWrapper{file}
	zipReader, err := zip.NewReader(w, size)
	if err != nil {
		return nil, fmt.Errorf("error opening zip file reader for history segment:%w", err)
	}

	for _, zipEntry := range zipReader.File {
		if filepath.Base(zipEntry.Name) == historyFileName {
			reader, err := zipEntry.Open()
			if err != nil {
				return nil, fmt.Errorf("error opening table csv file inside zip segment:%w", err)
			}
			return reader, nil
		}
	}

	return nil, fmt.Errorf("table file '%s' not found in segment", historyFileName)
}
