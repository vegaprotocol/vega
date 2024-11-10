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

package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/georgysavva/scany/pgxscan"
)

type (
	Vault struct {
		*ConnectionSource
	}
)

var vaultsOrdering = TableOrdering{
	ColumnOrdering{Name: "vault_id", Sorting: ASC},
}

func NewVault(connectionSource *ConnectionSource) *Vault {
	return &Vault{
		ConnectionSource: connectionSource,
	}
}

func (vs *Vault) Add(ctx context.Context, vaultState *entities.VaultState) error {
	defer metrics.StartSQLQuery("VaultState", "Add")()

	for _, v := range vaultState.PartyShares {
		_, err := vs.Exec(ctx,
			`INSERT INTO vault_party_shares(vault_id, party_id, share, vega_time)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (vega_time, vault_id, party_id) DO NOTHING`,
			v.VaultID,
			v.PartyID,
			v.Share,
			vaultState.VegaTime,
		)
		if err != nil {
			return err
		}
	}

	_, err := vs.Exec(ctx,
		`INSERT INTO vault_state(vault_id, vault, invested_amount, status, next_fee_calc, next_redemption_date, vega_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (vega_time, vault_id) DO NOTHING`,
		vaultState.VaultID,
		vaultState.Vault,
		vaultState.InvestedAmount,
		vaultState.Status,
		vaultState.NextFeeCalc,
		vaultState.NextRedemptionDate,
		vaultState.VegaTime,
	)

	return err
}

func (vs *Vault) GetByVaultID(
	ctx context.Context, id string,
) (entities.VaultState, error) {
	defer metrics.StartSQLQuery("Vaults", "GetByVaultID")()

	vaultState := entities.VaultState{}
	pvs := []*entities.VaultPartyShare{}
	err := pgxscan.Select(ctx, vs.ConnectionSource, &pvs,
		`SELECT * FROM vault_party_shares_current WHERE vault_id=$1`,
		entities.VaultID(id))
	if err != nil {
		return vaultState, vs.wrapE(err)
	}

	err = pgxscan.Get(ctx, vs.ConnectionSource, &vaultState,
		`SELECT * FROM vault_state_current WHERE vault_id=$1`,
		entities.VaultID(id))
	if err != nil {
		return vaultState, vs.wrapE(err)
	}
	vaultState.PartyShares = pvs
	return vaultState, vs.wrapE(err)
}

func (v *Vault) ListVaultsWithCursor(ctx context.Context,
	vaultIDs []string,
	assetIDs []string,
	liveOnly bool,
	pagination entities.CursorPagination,
) ([]entities.VaultState, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Vaults", "List Vaults")()
	vaultState := []entities.VaultState{}
	var pageInfo entities.PageInfo
	query := `SELECT * FROM vault_state_current`

	args := []interface{}{}
	query, args = addVaultWhereClause(query, args, vaultIDs, assetIDs, liveOnly)

	query, args, err := PaginateQuery[entities.VaultCursor](query, args, vaultsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, v.ConnectionSource, &vaultState, query, args...); err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("querying vault state: %w", err)
	}

	vaultStateWithPartyShare := make([]entities.VaultState, 0, len(vaultState))
	for _, vault := range vaultState {
		pvs := []*entities.VaultPartyShare{}
		err = pgxscan.Select(ctx, v.ConnectionSource, &pvs,
			`SELECT * FROM vault_party_shares_current WHERE vault_id=$1`, vault.VaultID)
		if err != nil {
			return nil, pageInfo, v.wrapE(err)
		}
		vault.PartyShares = pvs
		vaultStateWithPartyShare = append(vaultStateWithPartyShare, vault)
	}
	vaultStateWithPartyShare, pageInfo = entities.PageEntities(vaultStateWithPartyShare, pagination)
	return vaultStateWithPartyShare, pageInfo, nil
}

func prepareInClause(ids []string) ([]interface{}, string) {
	var args []interface{}
	var list strings.Builder
	for i, id := range ids {
		if i > 0 {
			list.WriteString(",")
		}

		list.WriteString(nextBindVar(&args, id))
	}
	return args, list.String()
}

func addVaultWhereClause(query string, args []interface{}, vaultIDs, assets []string, liveOnly bool) (string, []interface{}) {
	predicates := []string{}

	if len(vaultIDs) > 0 {
		inArgs, inList := prepareInClauseList[entities.VaultID](vaultIDs)
		args = append(args, inArgs...)
		predicates = append(predicates, fmt.Sprintf("vault_id IN (%s)", inList))
	}

	if len(assets) > 0 {
		inArgs, inList := prepareInClause(assets)
		args = append(args, inArgs...)
		predicates = append(predicates, fmt.Sprintf("vault->>'asset' IN (%s)", inList))
	}

	if liveOnly {
		predicates = append(predicates, "status!='VAULT_STATUS_STOPPED'")
	}

	if len(predicates) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(predicates, " AND "))
	}

	return query, args
}
