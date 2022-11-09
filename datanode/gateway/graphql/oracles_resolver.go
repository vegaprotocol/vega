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
	"errors"
	"strconv"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type oracleSpecResolver VegaResolverRoot

func (o *oracleSpecResolver) DataSourceSpec(ctx context.Context, obj *vegapb.OracleSpec) (*ExternalDataSourceSpec, error) {
	if obj.ExternalDataSourceSpec == nil {
		return nil, nil
	}

	dataSourceSpec := &DataSourceSpec{}
	if obj.ExternalDataSourceSpec.Spec != nil {
		if obj.ExternalDataSourceSpec.Spec.Data != nil {
			updatedAt := strconv.FormatInt(obj.ExternalDataSourceSpec.Spec.UpdatedAt, 10)
			dataSourceSpec.ID = obj.ExternalDataSourceSpec.Spec.Id
			dataSourceSpec.CreatedAt = strconv.FormatInt(obj.ExternalDataSourceSpec.Spec.CreatedAt, 10)
			dataSourceSpec.UpdatedAt = &updatedAt
			dataSourceSpec.Status = DataSourceSpecStatus(obj.ExternalDataSourceSpec.String())

			dataSourceSpec.Data = &DataSourceDefinition{
				SourceType: obj.ExternalDataSourceSpec.Spec.Data.SourceType.(DataSourceKind),
			}
		}
	}
	return &ExternalDataSourceSpec{
		Spec: dataSourceSpec,
	}, nil
}

func (o *oracleSpecResolver) DataConnection(ctx context.Context, obj *vegapb.OracleSpec, pagination *v2.Pagination) (*v2.OracleDataConnection, error) {
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

func (o *oracleDataResolver) ExternalData(ctx context.Context, obj *vegapb.OracleData) (*ExternalData, error) {
	if obj != nil {
		var signers []*Signer
		if len(obj.ExternalData.Data.Signers) > 0 {
			signers = make([]*Signer, len(obj.ExternalData.Data.Signers))
			for i := range obj.ExternalData.Data.Signers {
				signerObj, signer := obj.ExternalData.Data.Signers[i], &Signer{}
				if pubKey := signerObj.GetPubKey(); pubKey != nil {
					signer.Signer = &PubKey{Key: &pubKey.Key}
				} else if ethAddress := signerObj.GetEthAddress(); ethAddress != nil {
					signer.Signer = &ETHAddress{Address: &ethAddress.Address}
				} else {
					return nil, errors.New("invalid signer type")
				}
				signers[i] = signer
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
