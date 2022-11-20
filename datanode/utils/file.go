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
