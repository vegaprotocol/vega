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

	"code.vegaprotocol.io/vega/cmd/vega/commands/query"
	"github.com/jessevdk/go-flags"
)

type QueryCmd struct {
	Accounts          query.AccountsCmd          `command:"accounts" description:"Query a vega node to get the state of accounts"`
	Assets            query.AssetsCmd            `command:"assets" description:"Query a vega node to get the list of available assets"`
	NetworkParameters query.NetworkParametersCmd `command:"netparams" description:"Query a vega node to get the list network parameters"`
	Parties           query.PartiesCmd           `command:"parties" description:"Query a vega node to get the list of parties"`
	Validators        query.ValidatorsCmd        `command:"validators" description:"Query a vega node to get the list of the validators"`
	Markets           query.MarketsCmd           `command:"markets" description:"Query a vega node to get the list of all markets"`
	Proposals         query.ProposalsCmd         `command:"proposals" description:"Query a vega node to get the list of all proposals"`
	Help              bool                       `short:"h" long:"help" description:"Show this help message"`
}

var queryCmd QueryCmd

func Query(ctx context.Context, parser *flags.Parser) error {
	queryCmd = QueryCmd{}

	_, err := parser.AddCommand("query", "query state from a vega node", "", &queryCmd)
	return err
}
