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
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type oracleSpecResolver VegaResolverRoot

func (o *oracleSpecResolver) DataSourceSpec(ctx context.Context, obj *vegapb.OracleSpec) (*ExternalDataSourceSpec, error) {
	dataSourceSpec := &DataSourceSpec{
		Data: &DataSourceDefinition{},
	}

	if ed := obj.ExternalDataSourceSpec; ed != nil && ed.Spec != nil && ed.Spec.Data != nil {
		dataSourceSpec = resolveDataSourceSpec(ed.Spec)
	}

	return &ExternalDataSourceSpec{
		Spec: dataSourceSpec,
	}, nil
}

func (o *oracleSpecResolver) DataConnection(ctx context.Context, obj *vegapb.OracleSpec, pagination *v2.Pagination) (*v2.OracleDataConnection, error) {
	var specID *string
	if ed := obj.ExternalDataSourceSpec; ed != nil && ed.Spec != nil && ed.Spec.Id != "" {
		specID = &ed.Spec.Id
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
	ed := &ExternalData{
		Data: &Data{},
	}

	if obj != nil {
		if obj.ExternalData != nil && obj.ExternalData.Data != nil {
			o := obj.ExternalData.Data
			broadcastAt := strconv.FormatInt(o.BroadcastAt, 10)

			signers, err := resolveSigners(o.Signers)
			if err != nil {
				return nil, err
			}

			ed.Data = &Data{
				Signers:        signers,
				Data:           o.Data,
				MatchedSpecIds: o.MatchedSpecIds,
				BroadcastAt:    broadcastAt,
			}
		}
	}

	return ed, nil
}

func resolveSigners(obj []*v1.Signer) ([]*Signer, error) {
	signers := []*Signer{}
	if len(obj) > 0 {
		for i := range obj {
			o, signer := obj[i], &Signer{}

			if pk := o.GetPubKey(); pk != nil {
				signer.Signer = &PubKey{
					Key: &pk.Key,
				}
			} else if ethAddr := o.GetEthAddress(); ethAddr != nil {
				signer.Signer = &ETHAddress{
					Address: &ethAddr.Address,
				}
			} else {
				return signers, errors.New("invalid signer type")
			}

			signers = append(signers, signer)
		}
	}

	return signers, nil
}

func resolveDataSourceDefinition(d *vegapb.DataSourceDefinition) *DataSourceDefinition {
	ds := &DataSourceDefinition{}
	switch dst := d.SourceType.(type) {
	case *vegapb.DataSourceDefinition_External:
		if dst.External != nil {
			ds.SourceType = DataSourceDefinitionExternal{
				SourceType: dst.External.GetOracle(),
			}
		}
	case *vegapb.DataSourceDefinition_Internal:
		if dst.Internal != nil {
			ds.SourceType = DataSourceDefinitionInternal{
				SourceType: dst.Internal.GetTime(),
			}
		}
	}

	return ds
}

func resolveDataSourceSpec(d *vegapb.DataSourceSpec) *DataSourceSpec {
	ds := &DataSourceSpec{}

	if d != nil {
		updatedAt := strconv.FormatInt(d.UpdatedAt, 10)

		ds = &DataSourceSpec{
			ID:        d.GetId(),
			CreatedAt: strconv.FormatInt(d.CreatedAt, 10),
			UpdatedAt: &updatedAt,
			Status:    DataSourceSpecStatus(strconv.FormatInt(int64(d.Status), 10)),
		}

		if d.Data != nil {
			ds.Data = resolveDataSourceDefinition(d.Data)
		}
	}

	return ds
}
