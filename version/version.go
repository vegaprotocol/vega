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

package version

import (
	"runtime/debug"
	"strings"
)

var (
	cliVersionHash = ""
	cliVersion     = "v0.74.2"
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
