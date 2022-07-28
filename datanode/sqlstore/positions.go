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

package sqlstore

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

var ErrPositionNotFound = errors.New("party not found")

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
	return ps.batcher.Flush(ctx, ps.pool)
}

func (ps *Positions) Add(ctx context.Context, p entities.Position) error {
	ps.batcher.Add(p)
	return nil
}

func (ps *Positions) GetByMarketAndParty(ctx context.Context,
	marketID entities.MarketID,
	partyID entities.PartyID,
) (entities.Position, error) {
	position := entities.Position{}

	defer metrics.StartSQLQuery("Positions", "GetByMarketAndParty")()
	err := pgxscan.Get(ctx, ps.Connection, &position,
		`SELECT * FROM positions_current WHERE market_id=$1 AND party_id=$2`,
		marketID, partyID)

	if pgxscan.NotFound(err) {
		return position, fmt.Errorf("'%v/%v': %w", marketID, partyID, ErrPositionNotFound)
	}

	return position, err
}

func (ps *Positions) GetByMarket(ctx context.Context, marketID entities.MarketID) ([]entities.Position, error) {
	defer metrics.StartSQLQuery("Positions", "GetByMarket")()
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.Connection, &positions,
		`SELECT * FROM positions_current WHERE market_id=$1`,
		marketID)
	return positions, err
}

func (ps *Positions) GetByParty(ctx context.Context, partyID entities.PartyID) ([]entities.Position, error) {
	defer metrics.StartSQLQuery("Positions", "GetByParty")()
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.Connection, &positions,
		`SELECT * FROM positions_current WHERE party_id=$1`,
		partyID)
	return positions, err
}

func (ps *Positions) GetByPartyConnection(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID, pagination entities.CursorPagination) ([]entities.Position, entities.PageInfo, error) {
	var args []interface{}
	var pageInfo entities.PageInfo

	query := `select * from positions_current`
	var where string

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	positionCursor := &entities.PositionCursor{}
	if err := positionCursor.Parse(cursor); err != nil {
		return nil, pageInfo, fmt.Errorf("invalid cursor: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("vega_time", sorting, cmp, positionCursor.VegaTime),
	}

	if marketID.String() == "" {
		where = fmt.Sprintf(" where party_id=%s", nextBindVar(&args, partyID))
		cursorParams = append(cursorParams, NewCursorQueryParameter("market_id", sorting, cmp,
			entities.NewMarketID(positionCursor.MarketID)))
	} else {
		where = fmt.Sprintf(" where party_id=%s and market_id=%s", nextBindVar(&args, partyID), nextBindVar(&args, marketID))
	}

	query = fmt.Sprintf("%s %s", query, where)
	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	var positions []entities.Position
	if err := pgxscan.Select(ctx, ps.Connection, &positions, query, args...); err != nil {
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
