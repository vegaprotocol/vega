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
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
)

type StakeLinking struct {
	*ConnectionSource
}

const (
	sqlStakeLinkingColumns = `id, stake_linking_type, ethereum_timestamp, party_id, amount, stake_linking_status, finalized_at,
tx_hash, log_index, ethereum_address, vega_time`
)

func NewStakeLinking(connectionSource *ConnectionSource) *StakeLinking {
	return &StakeLinking{
		ConnectionSource: connectionSource,
	}
}

func (s *StakeLinking) Upsert(ctx context.Context, stake *entities.StakeLinking) error {
	defer metrics.StartSQLQuery("StakeLinking", "Upsert")()
	query := fmt.Sprintf(`insert into stake_linking (%s)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
on conflict (id, vega_time) do update
set
	stake_linking_type=EXCLUDED.stake_linking_type,
	ethereum_timestamp=EXCLUDED.ethereum_timestamp,
	party_id=EXCLUDED.party_id,
	amount=EXCLUDED.amount,
	stake_linking_status=EXCLUDED.stake_linking_status,
	finalized_at=EXCLUDED.finalized_at,
	tx_hash=EXCLUDED.tx_hash,
	log_index=EXCLUDED.log_index,
	ethereum_address=EXCLUDED.ethereum_address`, sqlStakeLinkingColumns)

	if _, err := s.Connection.Exec(ctx, query, stake.ID, stake.StakeLinkingType, stake.EthereumTimestamp, stake.PartyID, stake.Amount,
		stake.StakeLinkingStatus, stake.FinalizedAt, stake.TxHash, stake.LogIndex,
		stake.EthereumAddress, stake.VegaTime); err != nil {
		return err
	}

	return nil
}

func (s *StakeLinking) GetStake(ctx context.Context, partyID entities.PartyID,
	p entities.Pagination,
) (*num.Uint, []entities.StakeLinking, entities.PageInfo, error) {
	switch pagination := p.(type) {
	case entities.OffsetPagination:
		return s.getStakeWithOffsetPagination(ctx, partyID, pagination)
	case entities.CursorPagination:
		return s.getStakeWithCursorPagination(ctx, partyID, pagination)
	default:
		return nil, nil, entities.PageInfo{}, errors.New("invalid pagination provided")
	}
}

func (s *StakeLinking) getStakeWithOffsetPagination(ctx context.Context, partyID entities.PartyID, pagination entities.OffsetPagination) (
	*num.Uint, []entities.StakeLinking, entities.PageInfo, error) {
	var links []entities.StakeLinking
	var pageInfo entities.PageInfo
	// get the links from the database
	query, bindVars := getStakeLinkingQuery(partyID)
	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)

	var bal *num.Uint
	var err error

	defer metrics.StartSQLQuery("StakeLinking", "GetStake")()
	err = pgxscan.Select(ctx, s.Connection, &links, query, bindVars...)
	if err != nil {
		s.log.Errorf("could not retrieve links", logging.Error(err))
		return bal, nil, pageInfo, err
	}

	bal, err = s.calculateBalance(ctx, partyID)
	if err != nil {
		s.log.Errorf("cannot calculate balance", logging.Error(err))
		return num.Zero(), nil, pageInfo, err
	}
	return bal, links, pageInfo, nil
}

func (s *StakeLinking) getStakeWithCursorPagination(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) (
	*num.Uint, []entities.StakeLinking, entities.PageInfo, error) {
	var links []entities.StakeLinking
	var pageInfo entities.PageInfo
	// get the links from the database
	query, bindVars := getStakeLinkingQuery(partyID)

	sorting, cmp, cursor := extractPaginationInfo(pagination)
	sc := &entities.StakeLinkingCursor{}
	err := sc.Parse(cursor)
	if err != nil {
		return nil, nil, pageInfo, fmt.Errorf("could not parse pagination: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("vega_time", sorting, cmp, sc.VegaTime),
		NewCursorQueryParameter("id", sorting, cmp, entities.NewStakeLinkingID(sc.ID)),
	}

	query, bindVars = orderAndPaginateWithCursor(query, pagination, cursorParams, bindVars...)
	defer metrics.StartSQLQuery("StakeLinking", "GetStake")()

	var bal *num.Uint

	err = pgxscan.Select(ctx, s.Connection, &links, query, bindVars...)
	if err != nil {
		s.log.Errorf("could not retrieve links", logging.Error(err))
		return bal, nil, pageInfo, err
	}

	links, pageInfo = entities.PageEntities[*v2.StakeLinkingEdge](links, pagination)

	bal, err = s.calculateBalance(ctx, partyID)
	if err != nil {
		s.log.Errorf("cannot calculate balance", logging.Error(err))
		return num.Zero(), nil, pageInfo, err
	}
	return bal, links, pageInfo, nil
}

func getStakeLinkingQuery(partyID entities.PartyID) (string, []interface{}) {
	var bindVars []interface{}

	query := fmt.Sprintf(`select %s
from stake_linking_current
where party_id=%s`, sqlStakeLinkingColumns, nextBindVar(&bindVars, partyID))

	return query, bindVars
}

func (s *StakeLinking) calculateBalance(ctx context.Context, partyID entities.PartyID) (*num.Uint, error) {
	bal := num.Zero()
	var bindVars []interface{}

	query := fmt.Sprintf(`select coalesce(sum(CASE stake_linking_type
    WHEN 'TYPE_LINK' THEN amount
    WHEN 'TYPE_UNLINK' THEN -amount
    ELSE 0
    END), 0)
    FROM stake_linking_current
WHERE party_id = %s
  AND stake_linking_status = 'STATUS_ACCEPTED'
`, nextBindVar(&bindVars, partyID))

	var currentBalance decimal.Decimal
	defer metrics.StartSQLQuery("StakeLinking", "calculateBalance")()
	if err := pgxscan.Get(ctx, s.Connection, &currentBalance, query, bindVars...); err != nil {
		return bal, err
	}

	if currentBalance.LessThan(decimal.Zero) {
		return bal, errors.New("unlinked amount is greater than linked amount, potential missed events")
	}

	var overflowed bool
	if bal, overflowed = num.UintFromDecimal(currentBalance); overflowed {
		return num.Zero(), fmt.Errorf("current balance is invalid: %s", currentBalance.String())
	}

	return bal, nil
}
