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

package utils

import (
	"compress/gzip"
	"io"
	"os"
)

func CompressFile(source string, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	fileToWrite, err := os.Create(target)
	if err != nil {
		return err
	}

	zw := gzip.NewWriter(fileToWrite)
	if err != nil {
		return err
	}
	defer func() {
		_ = zw.Close()
	}()

	if _, err = io.Copy(zw, sourceFile); err != nil {
		return err
	}

	return nil
}

func DecompressFile(source string, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	fileToWrite, err := os.Create(target)
	if err != nil {
		return err
	}

	zr, err := gzip.NewReader(sourceFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = zr.Close()
	}()

	if _, err = io.Copy(fileToWrite, zr); err != nil {
		return err
	}

	return nil
}
