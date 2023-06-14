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

package query

import (
	"fmt"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/golang/protobuf/jsonpb"
)

type MarketsCmd struct {
	NodeAddress string `default:"0.0.0.0:3002" description:"The address of the vega node to use" long:"node-address"`
}

func (opts *MarketsCmd) Execute(_ []string) error {
	req := apipb.ListMarketsRequest{}
	return getPrintMarkets(opts.NodeAddress, &req)
}

func getPrintMarkets(nodeAddress string, req *apipb.ListMarketsRequest) error {
	clt, err := getClient(nodeAddress)
	if err != nil {
		return fmt.Errorf("could not connect to the vega node: %w", err)
	}

	ctx, cancel := timeoutContext()
	defer cancel()
	res, err := clt.ListMarkets(ctx, req)
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
