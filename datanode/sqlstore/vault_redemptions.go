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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/georgysavva/scany/pgxscan"
)

type (
	VaultRedemptions struct {
		*ConnectionSource
	}
)

var vaultsRedemptionsOrdering = TableOrdering{
	ColumnOrdering{Name: "request_id", Sorting: ASC},
}

func NewVaultRedemptions(connectionSource *ConnectionSource) *VaultRedemptions {
	return &VaultRedemptions{
		ConnectionSource: connectionSource,
	}
}

func (vr *VaultRedemptions) Add(ctx context.Context, request *entities.RedemptionRequest) error {
	defer metrics.StartSQLQuery("VaultRedemptions", "Add")()
	_, err := vr.Exec(ctx,
		`INSERT INTO vault_redemption_request(request_id, vault_id, party_id, asset, requested_amount, remaining_amount, eligibility_date, last_updated, status, vega_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (vega_time, request_id) DO NOTHING`,
		request.RequestID,
		request.VaultID,
		request.PartyID,
		request.Asset,
		request.RequestedAmount,
		request.RemainingAmount,
		request.EligibilityDate,
		request.LastUpdated,
		request.Status,
		request.VegaTime,
	)

	return err
}

func (vr *VaultRedemptions) GetByRequestID(
	ctx context.Context, id string,
) (entities.RedemptionRequest, error) {
	defer metrics.StartSQLQuery("VaultRedemptions", "GetByRequestID")()

	rr := entities.RedemptionRequest{}
	err := pgxscan.Get(ctx, vr.ConnectionSource, &rr,
		`SELECT * FROM vault_redemption_request_current WHERE request_id=$1`,
		entities.RedemptionRequestID(id))
	return rr, vr.wrapE(err)
}

func (vr *VaultRedemptions) ListRedemptionRequestsWithCursor(ctx context.Context, vaultIDs, partyIDs, assetIDs []string, statuses []vega.RedeemStatus, pagination entities.CursorPagination) ([]entities.RedemptionRequest, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("VaultRedemptions", "List Redemption Requests")()
	redemptionRequests := []entities.RedemptionRequest{}
	var pageInfo entities.PageInfo
	query := `SELECT * FROM vault_redemption_request_current`

	args := []interface{}{}
	query, args = addRedemptionRequestsWhereClause(query, args, vaultIDs, partyIDs, assetIDs, statuses)

	query, args, err := PaginateQuery[entities.RedemptionRequestCursor](query, args, vaultsRedemptionsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, vr.ConnectionSource, &redemptionRequests, query, args...); err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("querying vault state: %w", err)
	}

	redemptionRequests, pageInfo = entities.PageEntities(redemptionRequests, pagination)
	return redemptionRequests, pageInfo, nil
}

func prepareInClauseListMultiPred[A any, T entities.ID[A]](ids []string, args *[]interface{}) string {
	var list strings.Builder
	for i, id := range ids {
		if i > 0 {
			list.WriteString(",")
		}

		list.WriteString(nextBindVar(args, T(id)))
	}
	return list.String()
}

func addRedemptionRequestsWhereClause(query string, args []interface{}, vaultIDs, partyIDs, assets []string, statuses []vega.RedeemStatus) (string, []interface{}) {
	predicates := []string{}

	if len(vaultIDs) > 0 {
		inList := prepareInClauseListMultiPred[entities.VaultID](vaultIDs, &args)
		predicates = append(predicates, fmt.Sprintf("vault_id IN (%s)", inList))
	}

	if len(partyIDs) > 0 {
		inList := prepareInClauseListMultiPred[entities.PartyID](partyIDs, &args)
		predicates = append(predicates, fmt.Sprintf("party_id IN (%s)", inList))
	}

	if len(assets) > 0 {
		inList := prepareInClauseListMultiPred[entities.AssetID](assets, &args)
		predicates = append(predicates, fmt.Sprintf("asset IN (%s)", inList))
	}

	if len(statuses) > 0 {
		statusStrings := make([]string, 0, len(statuses))
		for _, status := range statuses {
			statusStrings = append(statusStrings, `'`+status.String()+`'`)
		}
		pred := fmt.Sprintf("status IN (%s)", strings.Join(statusStrings, ","))
		predicates = append(predicates, pred)
	}

	if len(predicates) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(predicates, " AND "))
	}

	return query, args
}
