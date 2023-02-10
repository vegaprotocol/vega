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

// TODO make these functions more robust.
type Args []string

func (a Args) indexOf(name string) int {
	for i, arg := range a {
		if strings.Contains(arg, name) {
			return i
		}
	}
	return -1
}

func (a Args) Exists(name string) bool {
	return a.indexOf(name) != -1
}

// Set sets a new argument. Ignores if argument exists.
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

// ForceSet sets a new argument even if the argument currently exists.
func (a *Args) ForceSet(name, value string) bool {
	if name[0:2] != "--" {
		name = fmt.Sprintf("--%s", name)
	}

	*a = append(*a, name, value)

	return true
}

// GetFlagWithArg finds and returns a flag with it's argument.
// Returns nil if not found.
// Example: --home /path.
func (a Args) GetFlagWithArg(name string) []string {
	if name[0:2] != "--" {
		name = fmt.Sprintf("--%s", name)
	}

	i := a.indexOf(name)
	if i == -1 {
		return nil
	}

	// Check if there is a flag's paramater available
	if len(a) < i+2 {
		return nil
	}

	return a[i : i+2]
}
