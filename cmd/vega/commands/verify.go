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
