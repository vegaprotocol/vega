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

package visor

import (
	"fmt"
	"strings"
)

// TODO make these functions more robust
type Args []string

func (a Args) Exists(name string) bool {
	for _, arg := range a {
		if strings.Contains(arg, name) {
			return true
		}
	}

	return false
}

func (a *Args) Set(name, value string) bool {
	if a.Exists(name) {
		return false
	}

	if name[0:2] != "--" {
		name = fmt.Sprintf("--%s", name)
	}

	*a = append(*a, name, value)

	return true
}
