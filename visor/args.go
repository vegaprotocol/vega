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
