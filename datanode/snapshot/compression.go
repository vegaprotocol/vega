package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/logging"
)

func tarAndCompressSnapshot(snapshotsDir string, snapshot snapshot, databaseVersion int64) (int64, error) {
	compressedFilePath := filepath.Join(snapshotsDir, snapshot.CompressedFileName())
	targetFile, err := os.Create(compressedFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create target file %s: %w", compressedFilePath, err)
	}

	zr := gzip.NewWriter(targetFile)
	tw := tar.NewWriter(zr)

	sourceDir := filepath.Join(snapshotsDir, snapshot.UncompressedDataDir())
	err = filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk files: %w", err)
		}

		if fi.IsDir() {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return fmt.Errorf("failed to get tar file header information for file %s: %w", file, err)
		}

		// Just take the bare minimum attributes to ensure tar is the same always for the same data
		headerMod := &tar.Header{
			Name:     header.Name,
			Size:     header.Size,
			Mode:     int64(fs.ModePerm),
			Devmajor: databaseVersion,
		}

		if err := tw.WriteHeader(headerMod); err != nil {
			return fmt.Errorf("failed to write tar file header information for file %s: %w", file, err)
		}

		data, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", file, err)
		}
		if _, err := io.Copy(tw, data); err != nil {
			return fmt.Errorf("failed to copy source file data %s: %w", file, err)
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to walk directory:%w", err)
	}

	if err := tw.Close(); err != nil {
		return 0, fmt.Errorf("failed to close compressed target file %s :%w", compressedFilePath, err)
	}

	if err := zr.Close(); err != nil {
		return 0, fmt.Errorf("failed to close compressed target file %s :%w", compressedFilePath, err)
	}

	fileInfo, err := os.Stat(compressedFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get statistics of compressed file %s: %w", compressedFilePath, err)
	}

	return fileInfo.Size(), nil
}

type snapshot interface {
	UncompressedDataDir() string
	CompressedFileName() string
}

func decompressAndUntarSnapshot(log *logging.Logger, snapshotsDir string, snapshot snapshot) (databaseVersion int64, err error) {
	decompressedFilesDestination := filepath.Join(snapshotsDir, snapshot.UncompressedDataDir())

	err = os.Mkdir(decompressedFilesDestination, fs.ModePerm)
	if err != nil {
		return 0, fmt.Errorf("failed to make target diretory for uncompressed files %s: %w", decompressedFilesDestination, err)
	}

	compressedFilePath := filepath.Join(snapshotsDir, snapshot.CompressedFileName())
	sourceFile, err := os.Open(compressedFilePath)
	defer func() {
		err = sourceFile.Close()
		if err != nil {
			log.Errorf("failed to close file:%w", err)
		}
	}()
	if err != nil {
		return 0, fmt.Errorf("failed to create source file %s: %w", compressedFilePath, err)
	}

	zr, err := gzip.NewReader(sourceFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create zip file reader for file %s: %w", compressedFilePath, err)
	}

	tr := tar.NewReader(zr)

	databaseVersion = -1
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read source file: %w", err)
		}

		targetFilePath := filepath.Join(decompressedFilesDestination, header.Name)
		dbVersion := header.Devmajor
		if databaseVersion == -1 {
			databaseVersion = dbVersion
		} else {
			if dbVersion != databaseVersion {
				return 0, fmt.Errorf("database version of all files should be the same")
			}
		}

		fileToWrite, err := os.OpenFile(targetFilePath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
		if err != nil {
			return 0, fmt.Errorf("failed to open uncompressed targetFile file %s: %w", targetFilePath, err)
		}

		if _, err := io.Copy(fileToWrite, tr); err != nil {
			return 0, fmt.Errorf("failed to copy compressed data to uncompressed target file %s:%w", targetFilePath, err)
		}

		if err := fileToWrite.Close(); err != nil {
			return 0, fmt.Errorf("failed to close uncompressed target file %s:%w", targetFilePath, err)
		}
	}

	return databaseVersion, nil
}
