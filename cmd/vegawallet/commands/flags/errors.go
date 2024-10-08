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

package flags

import (
	"fmt"
	"strings"
)

type FlagError struct {
	message string
}

func (f FlagError) Error() string {
	return f.message
}

func MutuallyExclusiveError(n1, n2 string) error {
	return FlagError{
		message: fmt.Sprintf("--%s and --%s flags are mutually exclusive", n1, n2),
	}
}

func MustBeSpecifiedError(name string) error {
	return FlagError{
		message: fmt.Sprintf("--%s flag must be specified", name),
	}
}

func RequireLessThanFlagError(less, greater string) error {
	return FlagError{
		message: fmt.Sprintf("--%s flag must be greater than --%s", greater, less),
	}
}

func ArgMustBeSpecifiedError(name string) error {
	return FlagError{
		message: fmt.Sprintf("%s argument must be specified", name),
	}
}

func TooManyArgsError(names ...string) error {
	return FlagError{
		message: fmt.Sprintf("too many arguments specified, only expect: %v", strings.Join(names, ", ")),
	}
}

func OneOfFlagsMustBeSpecifiedError(n1, n2 string) error {
	return FlagError{
		message: fmt.Sprintf("--%s or --%s flags must be specified", n1, n2),
	}
}

func InvalidFlagFormatError(name string) error {
	return FlagError{
		message: fmt.Sprintf("--%s flag has not a valid format", name),
	}
}

func UnsupportedFlagValueError(name string, unsupported interface{}, supported []interface{}) error {
	return FlagError{
		message: fmt.Sprintf("--%s flag doesn't support value %s, only %v", name, unsupported, supported),
	}
}

func OneOfParentsFlagMustBeSpecifiedError(name string, parents ...string) error {
	var resultFmt string
	if len(parents) > 1 {
		fmtFlags := make([]string, len(parents))
		for i, pf := range parents {
			fmtFlags[i] = fmt.Sprintf("--%s", pf)
		}
		flagsFmt := strings.Join([]string{
			strings.Join(parents[0:len(fmtFlags)-1], ", "),
			parents[len(fmtFlags)-1],
		}, " or ")
		resultFmt = fmt.Sprintf("%s flags", flagsFmt)
	} else {
		resultFmt = fmt.Sprintf("--%s flag", parents[0])
	}

	return FlagError{
		message: fmt.Sprintf("--%s flag requires %s to be set", name, resultFmt),
	}
}

func MustBase64EncodedError(name string) error {
	return FlagError{
		message: fmt.Sprintf("--%s flag value must be base64-encoded", name),
	}
}
