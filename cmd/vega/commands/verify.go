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

package commands

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/vega/commands/verify"

	"github.com/jessevdk/go-flags"
)

type VerifyCmd struct {
	Asset   verify.AssetCmd   `command:"passet"  description:"verify the payload of an asset proposal"`
	Genesis verify.GenesisCmd `command:"genesis" description:"verify the appstate of a genesis file"`
}

var verifyCmd VerifyCmd

func Verify(ctx context.Context, parser *flags.Parser) error {
	verifyCmd = VerifyCmd{}

	_, err := parser.AddCommand("verify", "Verify Vega payloads or genesis appstate", "", &verifyCmd)
	return err
}
