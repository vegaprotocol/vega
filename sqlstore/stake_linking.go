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
	pagination entities.OffsetPagination,
) (*num.Uint, []entities.StakeLinking) {
	var links []entities.StakeLinking
	var bindVars []interface{}
	// get the links from the database
	query := fmt.Sprintf(`select %s
from stake_linking_current
where party_id=%s`, sqlStakeLinkingColumns, nextBindVar(&bindVars, partyID))

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	var bal *num.Uint
	var err error

	defer metrics.StartSQLQuery("StakeLinking", "GetStake")()
	err = pgxscan.Select(ctx, s.Connection, &links, query, bindVars...)
	if err != nil {
		s.log.Errorf("could not retrieve links", logging.Error(err))
		return bal, nil
	}

	bal, err = s.calculateBalance(ctx, partyID)
	if err != nil {
		s.log.Errorf("cannot calculate balance", logging.Error(err))
		return num.Zero(), nil
	}
	return bal, links
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
