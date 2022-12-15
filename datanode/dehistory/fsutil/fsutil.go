package fsutil

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
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

func MkdirAllIgnoringUMask(dir string) error {
	// This creates the dir but takes into account the process's umask when setting permission bits
	err := os.MkdirAll(dir, fs.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to make directory: %s, error: %w", dir, err)
	}

	// This changes the permissions ignoring the process's umask
	err = os.Chmod(dir, fs.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to modify directory permissions: %s, error: %w", dir, err)
	}

	return nil
}

// TarAndCompressDirWithDeterministicHeader the tar file needs to be identical across OS/Arch, testing shows that if the
// default header is used there can be differences based on which platform the tar is created on.
func TarAndCompressDirWithDeterministicHeader(sourceDir, targetFile string, devMajorVersion int64) (compressedFileSize int64, err error) {
	file, err := os.Create(targetFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create target file %s: %w", targetFile, err)
	}

	zr := gzip.NewWriter(file)

	if err = TarDirectoryWithDeterministicHeader(zr, devMajorVersion, sourceDir); err != nil {
		return 0, fmt.Errorf("failed to tar directory:%w", err)
	}

	if err = zr.Close(); err != nil {
		return 0, fmt.Errorf("failed to close compressed target file %s :%w", targetFile, err)
	}

	fileInfo, err := os.Stat(targetFile)
	if err != nil {
		return 0, fmt.Errorf("failed to get statistics of compressed file %s: %w", targetFile, err)
	}

	compressedFileSize = fileInfo.Size()

	return
}

func TarDirectoryWithDeterministicHeader(w io.Writer, devMajorVersion int64, sourceDir string) error {
	tw := tar.NewWriter(w)
	err := filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
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
			Devmajor: devMajorVersion,
		}

		if err := tw.WriteHeader(headerMod); err != nil {
			return fmt.Errorf("failed to write tar file header information for file %s: %w", file, err)
		}

		data, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", file, err)
		}
		if _, err = io.Copy(tw, data); err != nil {
			return fmt.Errorf("failed to copy source file data %s: %w", file, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory:%w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer:%w", err)
	}
	return nil
}

func GetSnapshotDatabaseVersion(snapshotFile string) (int64, error) {
	sourceFile, err := os.Open(snapshotFile)
	defer func() { _ = sourceFile.Close() }()
	if err != nil {
		return 0, fmt.Errorf("failed to open snapshot file %s: %w", snapshotFile, err)
	}

	zr, err := gzip.NewReader(sourceFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create zip file reader for file %s: %w", snapshotFile, err)
	}

	tr := tar.NewReader(zr)

	header, err := tr.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to read source file header: %w", err)
	}

	return header.Devmajor, nil
}

func DecompressAndUntarFile(compressedFilePath, decompressedFilesDestination string) error {
	err := os.Mkdir(decompressedFilesDestination, fs.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to make target diretory for uncompressed files %s: %w", decompressedFilesDestination, err)
	}

	sourceFile, err := os.Open(compressedFilePath)
	defer func() { _ = sourceFile.Close() }()
	if err != nil {
		return fmt.Errorf("failed to create source file %s: %w", compressedFilePath, err)
	}

	zr, err := gzip.NewReader(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to create zip file reader for file %s: %w", compressedFilePath, err)
	}

	err = UntarFile(zr, decompressedFilesDestination)
	if err != nil {
		return fmt.Errorf("failed to untar file %s: %w", compressedFilePath, err)
	}

	return nil
}

func UntarFile(source io.Reader, decompressedFilesDestination string) error {
	tr := tar.NewReader(source)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read source file: %w", err)
		}

		targetFilePath := filepath.Join(decompressedFilesDestination, header.Name)

		fileToWrite, err := os.OpenFile(targetFilePath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("failed to open uncompressed targetFile file %s: %w", targetFilePath, err)
		}

		if _, err := io.Copy(fileToWrite, tr); err != nil {
			return fmt.Errorf("failed to copy compressed data to uncompressed target file %s:%w", targetFilePath, err)
		}

		if err := fileToWrite.Close(); err != nil {
			return fmt.Errorf("failed to close uncompressed target file %s:%w", targetFilePath, err)
		}
	}
	return nil
}
