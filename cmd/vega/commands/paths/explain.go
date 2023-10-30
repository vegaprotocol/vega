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

package paths

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/paths"
)

var (
	ErrProvideAPathToExplain       = errors.New("please provide a path to explain")
	ErrProvideOnlyOnePathToExplain = errors.New("please provide only one path to explain")
)

type ExplainCmd struct{}

func (opts *ExplainCmd) Execute(args []string) error {
	if argsLen := len(args); argsLen == 0 {
		return ErrProvideAPathToExplain
	} else if argsLen > 1 {
		return ErrProvideOnlyOnePathToExplain
	}

	pathName := args[0]

	explanation, err := paths.Explain(pathName)
	if err != nil {
		return fmt.Errorf("couldn't explain path %s: %w", pathName, err)
	}

	fmt.Printf("%s\n", explanation)
	return nil
}
