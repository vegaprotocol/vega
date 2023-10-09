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
