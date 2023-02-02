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
	"strings"

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

func (ps *Positions) GetByMarketAndParties(ctx context.Context, marketIDRaw string, partyIDsRaw []string) ([]entities.Position, error) {
	marketID := entities.MarketID(marketIDRaw)
	partyIDs := make([]interface{}, 0, len(partyIDsRaw))
	in := make([]string, 0, len(partyIDsRaw))
	bindNum := 2
	for _, p := range partyIDsRaw {
		partyIDs = append(partyIDs, entities.PartyID(p))
		in = append(in, fmt.Sprintf("$%d", bindNum))
		bindNum++
	}
	bind := make([]interface{}, 0, len(in)+1)
	// set all bind vars
	bind = append(bind, marketID)
	bind = append(bind, partyIDs...)
	positions := []entities.Position{}
	// build the query
	q := fmt.Sprintf(`SELECT * FROM positions_current WHERE market_id = $1 AND party_id IN (%s)`, strings.Join(in, ", "))
	err := pgxscan.Select(ctx, ps.Connection, &positions, q, bind...)
	return positions, err
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

func stringToPartyID(s ...string) [][]byte {
	partyIDs := make([][]byte, 0, len(s))
	for _, v := range s {
		if v == "" {
			continue
		}
		id := entities.PartyID(v)
		bs, err := id.Bytes()
		if err != nil {
			continue
		}
		partyIDs = append(partyIDs, bs)
	}
	return partyIDs
}

func stringToMarketID(s ...string) [][]byte {
	marketIDs := make([][]byte, 0, len(s))
	for _, v := range s {
		if v == "" {
			continue
		}
		id := entities.MarketID(v)
		bs, err := id.Bytes()
		if err != nil {
			continue
		}
		marketIDs = append(marketIDs, bs)
	}
	return marketIDs
}

func (ps *Positions) GetByPartyConnection(ctx context.Context, partyIDRaw []string, marketIDRaw []string, pagination entities.CursorPagination) ([]entities.Position, entities.PageInfo, error) {
	var (
		args     []interface{}
		pageInfo entities.PageInfo
		query    = `select * from positions_current`
		where    string
		partyID  = stringToPartyID(partyIDRaw...)
		marketID = stringToMarketID(marketIDRaw...)
		err      error
	)

	if len(partyID) > 0 && len(marketID) == 0 {
		where = fmt.Sprintf(" where party_id = ANY(%s::bytea[])", nextBindVar(&args, partyID))
	} else if len(partyID) > 0 && len(marketID) > 0 {
		where = fmt.Sprintf(" where party_id = ANY(%s::bytea[]) and market_id = ANY(%s::bytea[])", nextBindVar(&args, partyID), nextBindVar(&args, marketID))
	} else if len(partyID) == 0 && len(marketID) > 0 {
		where = fmt.Sprintf(" where market_id = ANY(%s::bytea[])", nextBindVar(&args, marketID))
	}

	if where != "" {
		query = fmt.Sprintf("%s %s", query, where)
	}

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
