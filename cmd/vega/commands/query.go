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

	"code.vegaprotocol.io/vega/cmd/vega/commands/query"

	"github.com/jessevdk/go-flags"
)

type QueryCmd struct {
	Accounts          query.AccountsCmd          `command:"accounts"                   description:"Query a vega node to get the state of accounts"`
	Assets            query.AssetsCmd            `command:"assets"                     description:"Query a vega node to get the list of available assets"`
	NetworkParameters query.NetworkParametersCmd `command:"netparams"                  description:"Query a vega node to get the list network parameters"`
	Parties           query.PartiesCmd           `command:"parties"                    description:"Query a vega node to get the list of parties"`
	Validators        query.ValidatorsCmd        `command:"validators"                 description:"Query a vega node to get the list of the validators"`
	Markets           query.MarketsCmd           `command:"markets"                    description:"Query a vega node to get the list of all markets"`
	Proposals         query.ProposalsCmd         `command:"proposals"                  description:"Query a vega node to get the list of all proposals"`
	Help              bool                       `description:"Show this help message" long:"help"                                                         short:"h"`
}

var queryCmd QueryCmd

func Query(ctx context.Context, parser *flags.Parser) error {
	queryCmd = QueryCmd{}

	_, err := parser.AddCommand("query", "query state from a vega node", "", &queryCmd)
	return err
}
