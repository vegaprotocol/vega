// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"code.vegaprotocol.io/vega/core/examples/nullchain/config"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
)

type Connection struct {
	conn     *grpc.ClientConn
	core     api.CoreServiceClient
	datanode v2.TradingDataServiceClient
	timeout  time.Duration
}

func NewConnection() (*Connection, error) {
	conn, err := grpc.Dial(config.GRCPAddress, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Connection{
		conn:     conn,
		core:     api.NewCoreServiceClient(conn),
		datanode: v2.NewTradingDataServiceClient(conn),
		timeout:  5 * time.Second,
	}, nil
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) LastBlockHeight() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	bhReq := &api.LastBlockHeightRequest{}
	resp, err := c.core.LastBlockHeight(ctx, bhReq)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return resp.Height, nil
}

func (c *Connection) NetworkChainID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	bhReq := &api.StatisticsRequest{}
	resp, err := c.core.Statistics(ctx, bhReq)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return resp.Statistics.ChainId, nil
}

func (c *Connection) VegaTime() (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	gvtReq := &v2.GetVegaTimeRequest{}
	response, err := c.datanode.GetVegaTime(ctx, gvtReq)
	if err != nil {
		return time.Time{}, errors.WithStack(err)
	}

	t := time.Unix(0, response.Timestamp)
	return t, nil
}

func (c *Connection) GetProposalsByParty(party *Party) ([]*vega.GovernanceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	r, err := c.datanode.ListGovernanceData(ctx,
		&v2.ListGovernanceDataRequest{
			ProposerPartyId: &party.pubkey,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var proposals []*vega.GovernanceData
	for _, gd := range r.GetConnection().GetEdges() {
		proposals = append(proposals, gd.GetNode())
	}

	return proposals, nil
}

func (c *Connection) GetProposalByReference(ref string) (*vega.Proposal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	r, err := c.datanode.GetGovernanceData(ctx,
		&v2.GetGovernanceDataRequest{
			Reference: &ref,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return r.Data.Proposal, nil
}

func (c *Connection) GetMarkets() ([]*vega.Market, error) {
	r, err := c.datanode.ListMarkets(context.Background(), &v2.ListMarketsRequest{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var markets []*vega.Market
	for _, m := range r.GetMarkets().GetEdges() {
		markets = append(markets, m.GetNode())
	}

	return markets, nil
}
