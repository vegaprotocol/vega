package cmd_test

import (
	"path/filepath"
	"testing"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

func NewTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func NewPassphraseFile(t *testing.T, path string) (string, string) {
	t.Helper()
	passphrase := vgrand.RandomStr(10)
	fuzz := vgrand.RandomStr(5)

	passphraseFilePath := NewFile(t, path, fuzz+"passphrase.txt", passphrase)
	return passphrase, passphraseFilePath
}

func NewFile(t *testing.T, path, fileName, data string) string {
	t.Helper()
	filePath := filepath.Join(path, fileName)
	if err := vgfs.WriteFile(filePath, []byte(data)); err != nil {
		t.Fatalf("couldn't write passphrase file: %v", err)
	}
	return filePath
}
