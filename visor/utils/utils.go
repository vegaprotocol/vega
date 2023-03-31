// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package utils

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func AbsPath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}

	aPath, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %q: %w", p, err)
	}

	return aPath, nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check if %q path exists: %w", path, err)
}

func EnsureBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		_, f := filepath.Split(path)
		return fmt.Errorf("%s is not a regular file", f)
	}
	// Make sure all executable bits are set.
	oldMode := info.Mode().Perm()
	newMode := oldMode | 0o111 // Set the three execute bits to on (a+x).
	if oldMode != newMode {
		return os.Chmod(path, newMode)
	}
	return nil
}

// UnzipSource iterates over zip files inside the archive and unzips each of them to the destination.
func UnzipSource(source, destination string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

func ToLookupMap[T comparable](slice []T) map[T]struct{} {
	m := make(map[T]struct{}, len(slice))
	for _, v := range slice {
		m[v] = struct{}{}
	}
	return m
}

func ExecuteBinary(binaryPath string, args []string, v interface{}) ([]byte, error) {
	command := exec.Command(binaryPath, args...)

	var stdOut, stErr bytes.Buffer
	command.Stdout = &stdOut
	command.Stderr = &stErr

	if err := command.Run(); err != nil {
		return nil, fmt.Errorf(
			"failed to execute binary %s %v with error: %s, %s: %s",
			binaryPath,
			args,
			stErr.String(),
			stdOut.String(),
			err.Error(),
		)
	}

	if v == nil {
		return stdOut.Bytes(), nil
	}

	if err := json.Unmarshal(stdOut.Bytes(), v); err != nil {
		return nil, fmt.Errorf("failed to parse command output %q: %w", stdOut.String(), err)
	}

	return nil, nil
}
