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

package genesis

import (
	"context"

	"github.com/jessevdk/go-flags"
)

type newCmd struct {
	Validator newValidatorCmd `command:"validator" description:"Show information to become validator"`
}

func initNewCmd(_ context.Context, parentCmd *flags.Command) error {
	cmd := newCmd{
		Validator: newValidatorCmd{
			TmHome: "$HOME/.cometbft",
		},
	}

	var (
		short = "Create a resource"
		long  = "Create a resource"
	)

	_, err := parentCmd.AddCommand("new", short, long, &cmd)
	return err
}
