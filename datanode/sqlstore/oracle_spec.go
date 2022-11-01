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

type OracleSpec struct {
	*ConnectionSource
}

var oracleSpecOrdering = TableOrdering{
	ColumnOrdering{Name: "id", Sorting: ASC},
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

const (
	sqlOracleSpecColumns = `id, created_at, updated_at, signers, filters, status, tx_hash, vega_time`
)

func NewOracleSpec(connectionSource *ConnectionSource) *OracleSpec {
	return &OracleSpec{
		ConnectionSource: connectionSource,
	}
}

func (os *OracleSpec) Upsert(ctx context.Context, spec *entities.OracleSpec) error {
	query := fmt.Sprintf(`insert into oracle_specs(%s)
values ($1, $2, $3, $4, $5, $6, $7, $8)
on conflict (id, vega_time) do update
set
	created_at=EXCLUDED.created_at,
	updated_at=EXCLUDED.updated_at,
	signers=EXCLUDED.signers,
	filters=EXCLUDED.filters,
	status=EXCLUDED.status,
	tx_hash=EXCLUDED.tx_hash`, sqlOracleSpecColumns)

	defer metrics.StartSQLQuery("OracleSpec", "Upsert")()
	dataSourceSpec := spec.ExternalDataSourceSpec.Spec
	signers := []entities.Signer{}
	filters := []entities.Filter{}
	if dataSourceSpec.Config != nil {
		filters = dataSourceSpec.Config.Filters
		signers = dataSourceSpec.Config.Signers
	}
	if _, err := os.Connection.Exec(ctx, query, dataSourceSpec.ID, dataSourceSpec.CreatedAt, dataSourceSpec.UpdatedAt, signers,
		filters, dataSourceSpec.Status, dataSourceSpec.TxHash, dataSourceSpec.VegaTime); err != nil {
		return err
	}

	return nil
}

func (os *OracleSpec) GetSpecByID(ctx context.Context, specID string) (*entities.OracleSpec, error) {
	var spec entities.DataSourceSpecRaw
	query := fmt.Sprintf(`%s
where id = $1
order by id, vega_time desc`, getOracleSpecsQuery())

	defer metrics.StartSQLQuery("OracleSpec", "GetByID")()
	err := pgxscan.Get(ctx, os.Connection, &spec, query, entities.SpecID(specID))
	return &entities.OracleSpec{
		ExternalDataSourceSpec: &entities.ExternalDataSourceSpec{
			Spec: &entities.DataSourceSpec{
				ID:        spec.ID,
				CreatedAt: spec.CreatedAt,
				UpdatedAt: spec.UpdatedAt,
				Config: &entities.DataSourceSpecConfiguration{
					Filters: spec.Filters,
					Signers: spec.Signers,
				},
				Status:   spec.Status,
				TxHash:   spec.TxHash,
				VegaTime: spec.VegaTime,
			},
		},
	}, err
}

func (os *OracleSpec) GetSpecs(ctx context.Context, pagination entities.OffsetPagination) ([]entities.DataSourceSpec, error) {
	var specsRaw []entities.DataSourceSpecRaw
	query := fmt.Sprintf(`%s order by id, vega_time desc`, getOracleSpecsQuery())

	var bindVars []interface{}
	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	defer metrics.StartSQLQuery("OracleSpec", "ListOracleSpecs")()
	err := pgxscan.Select(ctx, os.Connection, &specsRaw, query, bindVars...)

	specs := []entities.DataSourceSpec{}
	for _, specRaw := range specsRaw {
		specs = append(specs,
			entities.DataSourceSpec{
				ID:        specRaw.ID,
				CreatedAt: specRaw.CreatedAt,
				UpdatedAt: specRaw.UpdatedAt,
				Config: &entities.DataSourceSpecConfiguration{
					Filters: specRaw.Filters,
					Signers: specRaw.Signers,
				},
				Status:   specRaw.Status,
				TxHash:   specRaw.TxHash,
				VegaTime: specRaw.VegaTime,
			},
		)
	}
	return specs, err
}

func (os *OracleSpec) GetSpecsWithCursorPagination(ctx context.Context, specID string, pagination entities.CursorPagination) (
	[]entities.OracleSpec, entities.PageInfo, error,
) {
	if specID != "" {
		return os.getSingleSpecWithPageInfo(ctx, specID)
	}

	return os.getSpecsWithPageInfo(ctx, pagination)
}

func (os *OracleSpec) getSingleSpecWithPageInfo(ctx context.Context, specID string) ([]entities.OracleSpec, entities.PageInfo, error) {
	spec, err := os.GetSpecByID(ctx, specID)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return []entities.OracleSpec{*spec},
		entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     spec.Cursor().Encode(),
			EndCursor:       spec.Cursor().Encode(),
		}, nil
}

func (os *OracleSpec) getSpecsWithPageInfo(ctx context.Context, pagination entities.CursorPagination) (
	[]entities.OracleSpec, entities.PageInfo, error,
) {
	var (
		dataSpecs []entities.DataSourceSpecRaw
		specs     = []entities.OracleSpec{}
		pageInfo  entities.PageInfo
		err       error
		args      []interface{}
	)

	query := getOracleSpecsQuery()
	query, args, err = PaginateQuery[entities.DataSourceSpecCursor](query, args, oracleSpecOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err = pgxscan.Select(ctx, os.Connection, &dataSpecs, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("querying oracle specs: %w", err)
	}

	if len(dataSpecs) > 0 {
		for i := range dataSpecs {
			specs = append(specs, entities.OracleSpec{
				ExternalDataSourceSpec: &entities.ExternalDataSourceSpec{
					Spec: &entities.DataSourceSpec{
						ID:        dataSpecs[i].ID,
						CreatedAt: dataSpecs[i].CreatedAt,
						UpdatedAt: dataSpecs[i].UpdatedAt,
						Config: &entities.DataSourceSpecConfiguration{
							Filters: dataSpecs[i].Filters,
							Signers: dataSpecs[i].Signers,
						},
						Status:   dataSpecs[i].Status,
						TxHash:   dataSpecs[i].TxHash,
						VegaTime: dataSpecs[i].VegaTime,
					},
				},
			})
		}
	}
	specs, pageInfo = entities.PageEntities[*v2.OracleSpecEdge](specs, pagination)

	return specs, pageInfo, nil
}

func getOracleSpecsQuery() string {
	return fmt.Sprintf(`select distinct on (id) %s
from oracle_specs`, sqlOracleSpecColumns)
}
