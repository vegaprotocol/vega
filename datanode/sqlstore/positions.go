// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

var positionsOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "party_id", Sorting: ASC},
	ColumnOrdering{Name: "market_id", Sorting: ASC},
}

type Positions struct {
	*ConnectionSource
	batcher MapBatcher[entities.PositionKey, entities.Position]
}

func NewPositions(connectionSource *ConnectionSource) *Positions {
	a := &Positions{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.PositionKey, entities.Position](
			"positions",
			entities.PositionColumns),
	}
	return a
}

func (ps *Positions) Flush(ctx context.Context) ([]entities.Position, error) {
	defer metrics.StartSQLQuery("Positions", "Flush")()
	return ps.batcher.Flush(ctx, ps.Connection)
}

// AddBatch just allows you to add several entities in a single call
func (ps *Positions) AddBatch(ctx context.Context, pos ...entities.Position) error {
	for _, p := range pos {
		ps.batcher.Add(p)
	}
	return nil
}

func (ps *Positions) Add(ctx context.Context, p entities.Position) error {
	ps.batcher.Add(p)
	return nil
}

func (ps *Positions) GetByMarketAndParty(ctx context.Context,
	marketIDRaw string,
	partyIDRaw string,
) (entities.Position, error) {
	var (
		position = entities.Position{}
		marketID = entities.MarketID(marketIDRaw)
		partyID  = entities.PartyID(partyIDRaw)
	)

	defer metrics.StartSQLQuery("Positions", "GetByMarketAndParty")()
	return position, ps.wrapE(pgxscan.Get(ctx, ps.Connection, &position,
		`SELECT * FROM positions_current WHERE market_id=$1 AND party_id=$2`,
		marketID, partyID))
}

func (ps *Positions) GetByMarket(ctx context.Context, marketID string) ([]entities.Position, error) {
	defer metrics.StartSQLQuery("Positions", "GetByMarket")()
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.Connection, &positions,
		`SELECT * FROM positions_current WHERE market_id=$1`,
		entities.MarketID(marketID))
	return positions, err
}

func (ps *Positions) GetByParty(ctx context.Context, partyID string) ([]entities.Position, error) {
	defer metrics.StartSQLQuery("Positions", "GetByParty")()
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.Connection, &positions,
		`SELECT * FROM positions_current WHERE party_id=$1`,
		entities.PartyID(partyID))
	return positions, err
}

func (ps *Positions) GetByPartyConnection(ctx context.Context, partyIDRaw string, marketIDRaw string, pagination entities.CursorPagination) ([]entities.Position, entities.PageInfo, error) {
	var (
		args     []interface{}
		pageInfo entities.PageInfo
		query    = `select * from positions_current`
		where    string
		partyID  = entities.PartyID(partyIDRaw)
		marketID = entities.MarketID(marketIDRaw)
		err      error
	)

	if marketID == "" {
		where = fmt.Sprintf(" where party_id=%s", nextBindVar(&args, partyID))
	} else {
		where = fmt.Sprintf(" where party_id=%s and market_id=%s", nextBindVar(&args, partyID), nextBindVar(&args, marketID))
	}

	query = fmt.Sprintf("%s %s", query, where)
	query, args, err = PaginateQuery[entities.PositionCursor](query, args, positionsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	var positions []entities.Position
	if err = pgxscan.Select(ctx, ps.Connection, &positions, query, args...); err != nil {
		return nil, pageInfo, err
	}

	positions, pageInfo = entities.PageEntities[*v2.PositionEdge](positions, pagination)
	return positions, pageInfo, nil
}

func (ps *Positions) GetAll(ctx context.Context) ([]entities.Position, error) {
	defer metrics.StartSQLQuery("Positions", "GetAll")()
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.Connection, &positions,
		`SELECT * FROM positions_current`)
	return positions, err
}
