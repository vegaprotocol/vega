package tests_test

import (
	"fmt"
	"path/filepath"
	"testing"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

const testRecoveryPhrase = "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render"

func NewPassphraseFile(t *testing.T, path string) (string, string) {
	t.Helper()
	passphrase := vgrand.RandomStr(10)
	passphraseFilePath := NewFile(t, path, fmt.Sprintf("passphrase-%s.txt", passphrase), passphrase)
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

func FakeNetwork(name string) string {
	return fmt.Sprintf(`
Name = "%s"
Level = "info"
MaximumTokenDuration = "1h0m0s"
Port = 8000
Host = "127.0.0.1"

[API.GRPC]
Retries = 5
Hosts = [
    "example.com:3007",
]

[API.REST]
Hosts = [
    "https://example.com/rest"
]

[API.GraphQL]
Hosts = [
    "https://example.com/gql/query"
]
`, name)
}
