package nullchain

import (
	"context"
	"time"

	config "code.vegaprotocol.io/vega/examples/nullchain/config"
	"github.com/pkg/errors"

	datanode "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/protos/vega"
	api "code.vegaprotocol.io/protos/vega/api/v1"
	"google.golang.org/grpc"
)

type Connection struct {
	conn     *grpc.ClientConn
	core     api.CoreServiceClient
	datanode datanode.TradingDataServiceClient
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
		datanode: datanode.NewTradingDataServiceClient(conn),
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

func (c *Connection) VegaTime() (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	gvtReq := &datanode.GetVegaTimeRequest{}
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

	r, err := c.datanode.GetProposalsByParty(ctx,
		&datanode.GetProposalsByPartyRequest{
			PartyId: party.pubkey,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return r.Data, nil
}

func (c *Connection) GetProposalByReference(ref string) (*vega.Proposal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	r, err := c.datanode.GetProposalByReference(ctx,
		&datanode.GetProposalByReferenceRequest{
			Reference: ref,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return r.Data.Proposal, nil
}

func (c *Connection) GetMarkets() ([]*vega.Market, error) {
	markets, err := c.datanode.Markets(context.Background(), &datanode.MarketsRequest{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return markets.Markets, nil
}
