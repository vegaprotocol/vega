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
	"fmt"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/golang/protobuf/jsonpb"
)

type AssetsCmd struct {
	NodeAddress string `default:"0.0.0.0:3002" description:"The address of the vega node to use" long:"node-address"`
}

func (opts *AssetsCmd) Execute(_ []string) error {
	req := apipb.ListAssetsRequest{}
	return getPrintAssets(opts.NodeAddress, &req)
}

func getPrintAssets(nodeAddress string, req *apipb.ListAssetsRequest) error {
	clt, err := getClient(nodeAddress)
	if err != nil {
		return fmt.Errorf("could not connect to the vega node: %w", err)
	}

	ctx, cancel := timeoutContext()
	defer cancel()
	res, err := clt.ListAssets(ctx, req)
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
