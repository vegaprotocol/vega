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

package gql

import (
	"context"
	"strconv"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	v11 "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type oracleSpecResolver VegaResolverRoot

func (o *oracleSpecResolver) DataSourceSpec(ctx context.Context, obj *v1.OracleSpec) (*ExternalDataSourceSpec, error) {
	spec := obj.ExternalDataSourceSpec.Spec
	updatedAt := strconv.FormatInt(spec.UpdatedAt, 10)

	signers := []*v1.Signer{}
	filters := []*v1.Filter{}
	if spec.Config != nil {
		signers = spec.Config.Signers
		filters = spec.Config.Filters
	}

	ds := &DataSourceSpec{
		ID:        spec.Id,
		CreatedAt: strconv.FormatInt(spec.CreatedAt, 10),
		UpdatedAt: &updatedAt,
		Config: &v1.DataSourceSpecConfiguration{
			Signers: signers,
			Filters: filters,
		},
		Status: DataSourceSpecStatus(spec.Status.String()),
	}
	return &ExternalDataSourceSpec{
		Spec: ds,
	}, nil
}

func (o *oracleSpecResolver) DataConnection(ctx context.Context, obj *v11.OracleSpec, pagination *v2.Pagination) (*v2.OracleDataConnection, error) {
	var specID *string
	if obj != nil && obj.ExternalDataSourceSpec.Spec.Id != "" {
		specID = &obj.ExternalDataSourceSpec.Spec.Id
	}
	req := v2.ListOracleDataRequest{
		OracleSpecId: specID,
		Pagination:   pagination,
	}

	resp, err := o.tradingDataClientV2.ListOracleData(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.OracleData, nil
}

type oracleDataResolver VegaResolverRoot

func (o *oracleDataResolver) ExternalData(ctx context.Context, obj *v1.OracleData) (*ExternalData, error) {
	if obj != nil {
		var signers []*Signer
		if len(obj.ExternalData.Data.Signers) > 0 {
			signers = make([]*Signer, len(obj.ExternalData.Data.Signers))
			for i, signer := range obj.ExternalData.Data.Signers {
				signers[i] = &Signer{
					Signer: signer.GetSigner().(SignerKind),
				}
			}
		}
		ed := &ExternalData{
			Data: &Data{
				Signers:        signers,
				Data:           obj.ExternalData.Data.Data,
				MatchedSpecIds: obj.ExternalData.Data.MatchedSpecIds,
				BroadcastAt:    strconv.FormatInt(obj.ExternalData.Data.BroadcastAt, 10),
			},
		}
		return ed, nil
	}
	return nil, nil
}
