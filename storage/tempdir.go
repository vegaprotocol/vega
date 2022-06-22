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

package storage

import (
	"fmt"
	"io/ioutil"
	"os"
)

func noop() {
	// NO-OP
}

// TempDir will create a temporary folderwith the give prefix
func TempDir(prefix string) (string, func(), error) {
	baseTempDirs := []string{"/dev/shm", os.TempDir()}
	for _, baseTempDir := range baseTempDirs {
		_, err := os.Stat(baseTempDir)
		if err == nil {
			dir, err := ioutil.TempDir(baseTempDir, prefix)
			if err != nil {
				return "", noop, fmt.Errorf("could not create tmp dir in %s", baseTempDir)
			}
			return dir, func() { os.RemoveAll(dir) }, nil
		}
	}
	return "", noop, fmt.Errorf("could not find a temp dir")
}
