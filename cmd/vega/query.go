package main

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/vega/query"
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
