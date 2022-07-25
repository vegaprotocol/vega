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

import "runtime/debug"

var (
	cLIVersionHash = ""
	cLIVersion     = "v0.54.0+dev"
)

func init() {
	info, _ := debug.ReadBuildInfo()
	modified := false

	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			cLIVersionHash = v.Value
		}
		if v.Key == "vcs.modified" {
			modified = true
		}
	}
	if modified {
		cLIVersionHash += "-modified"
	}
}

func Get() string {
	return cLIVersion
}

func GetCommitHash() string {
	return cLIVersionHash
}
