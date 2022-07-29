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
