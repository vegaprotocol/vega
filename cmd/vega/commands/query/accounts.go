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

package query

import (
	"errors"
	"fmt"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/golang/protobuf/jsonpb"
)

type AccountsCmd struct {
	Party   AccountsPartyCmd   `command:"party"                      description:"List accounts for a given party"`
	Market  AccountsMarketCmd  `command:"market"                     description:"List accounts for a given market"`
	Network AccountsNetworkCmd `command:"network"                    description:"List accounts owned by the network"`
	Help    bool               `description:"Show this help message" long:"help"                                      short:"h"`
}

type AccountsPartyCmd struct {
	NodeAddress string `default:"0.0.0.0:3002"               description:"The address of the vega node to use" long:"node-address"`
	Market      string `description:"An optional market"     long:"market"`
	Help        bool   `description:"Show this help message" long:"help"                                       short:"h"`
}

type AccountsMarketCmd struct {
	NodeAddress string `default:"0.0.0.0:3002"               description:"The address of the vega node to use" long:"node-address"`
	Help        bool   `description:"Show this help message" long:"help"                                       short:"h"`
}

type AccountsNetworkCmd struct {
	NodeAddress string `default:"0.0.0.0:3002"               description:"The address of the vega node to use" long:"node-address"`
	Help        bool   `description:"Show this help message" long:"help"                                       short:"h"`
}

func (opts *AccountsPartyCmd) Execute(params []string) error {
	if len(params) > 1 {
		return errors.New("only one party needs to be specified")
	}

	if len(params) < 1 {
		return errors.New("one party is required")
	}

	req := apipb.ListAccountsRequest{
		Party:  params[0],
		Market: opts.Market, // at most empty anyway
	}

	return getPrintAccounts(opts.NodeAddress, &req)
}

func (opts *AccountsMarketCmd) Execute(params []string) error {
	if len(params) > 1 {
		return errors.New("only one market needs to be specified")
	}

	return nil
}

func (opts *AccountsNetworkCmd) Execute(_ []string) error {
	req := apipb.ListAccountsRequest{}
	return getPrintAccounts(opts.NodeAddress, &req)
}

func getPrintAccounts(nodeAddress string, req *apipb.ListAccountsRequest) error {
	clt, err := getClient(nodeAddress)
	if err != nil {
		return fmt.Errorf("could not connect to the vega node: %w", err)
	}

	ctx, cancel := timeoutContext()
	defer cancel()
	res, err := clt.ListAccounts(ctx, req)
	if err != nil {
		return fmt.Errorf("error querying the vega node: %w", err)
	}

	m := jsonpb.Marshaler{
		Indent: "  ",
	}
	buf, err := m.MarshalToString(res)
	if err != nil {
		return fmt.Errorf("invalid response from vega node: %w", err)
	}

	fmt.Printf("%v", buf)

	return nil
}
