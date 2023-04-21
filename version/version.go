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

package version

import (
	"runtime/debug"
	"strings"
)

var (
	cliVersionHash = ""
	cliVersion     = "v0.71.0"
)

func init() {
	info, _ := debug.ReadBuildInfo()
	// for some reason in jenkins integration tests this return nil
	if info == nil {
		cliVersionHash = "unknown"
		return
	}

	modified := false

	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			cliVersionHash = v.Value
		}
		if v.Key == "vcs.modified" && v.Value == "true" {
			modified = true
		}
	}
	if modified {
		cliVersionHash += "-modified"
	}
}

// Get returns the version of the software. When the version is a development
// version, the first 8 characters of the git hash, is appended as a build tag.
// Any dash separated addition to the hash is appended as a build tag as well.
func Get() string {
	finalVersion := cliVersion

	if strings.HasSuffix(cliVersion, "+dev") {
		splatHash := strings.Split(cliVersionHash, "-")

		// Verifying if splitting the version hash gave results.
		if len(splatHash) == 0 {
			return finalVersion
		}

		// Verifying if there is a commit hash.
		if splatHash[0] != "" {
			finalVersion = finalVersion + "." + splatHash[0][:8]
		}

		// Anything left from the splitting is appended as build tags behind.
		for i := 1; i < len(splatHash); i++ {
			finalVersion = finalVersion + "." + splatHash[i]
		}
	}
	return finalVersion
}

func GetCommitHash() string {
	return cliVersionHash
}
