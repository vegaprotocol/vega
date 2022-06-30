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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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
	var query string
	if marketID.String() != "" {
		query = fmt.Sprintf(`select * from positions_current where party_id=%s`, partyID.String())
	} else {
		query = fmt.Sprintf(`select * from positions_current where party_id=%s and market_id=%s`,
			partyID.String(), marketID.String())
	}

	positions := make([]entities.Position, 0)
	args := make([]interface{}, 0)

	var pagedPositions []entities.Position
	var pageInfo entities.PageInfo

	sorting, cmp, cursor := extractPaginationInfo(pagination)
	cursors := []CursorQueryParameter{NewCursorQueryParameter("vega_time", sorting, cmp, cursor)}
	query, args = orderAndPaginateWithCursor(query, pagination, cursors, args...)

	if err := pgxscan.Select(ctx, ps.Connection, &positions, query, args...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	pagedPositions, pageInfo = entities.PageEntities[*v2.PositionEdge](positions, pagination)
	return pagedPositions, pageInfo, nil
}

func (ps *Positions) GetAll(ctx context.Context) ([]entities.Position, error) {
	defer metrics.StartSQLQuery("Positions", "GetAll")()
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.Connection, &positions,
		`SELECT * FROM positions_current`)
	return positions, err
}
