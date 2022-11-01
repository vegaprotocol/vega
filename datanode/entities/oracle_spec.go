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

package entities

import (
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type OracleSpec struct {
	ExternalDataSourceSpec *ExternalDataSourceSpec
}

func OracleSpecFromProto(spec *datapb.OracleSpec, txHash TxHash, vegaTime time.Time) (*OracleSpec, error) {
	if spec.ExternalDataSourceSpec != nil {
		ds, err := ExternalDataSourceSpecFromProto(spec.ExternalDataSourceSpec, txHash, vegaTime)
		if err != nil {
			return nil, err
		}

		return &OracleSpec{
			ExternalDataSourceSpec: ds,
		}, nil
	}

	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{},
	}, nil
}

func (os *OracleSpec) ToProto() *datapb.OracleSpec {
	return &datapb.OracleSpec{
		ExternalDataSourceSpec: os.ExternalDataSourceSpec.ToProto(),
	}
}

func (os OracleSpec) Cursor() *Cursor {
	return NewCursor(DataSourceSpecCursor{os.ExternalDataSourceSpec.Spec.VegaTime, os.ExternalDataSourceSpec.Spec.ID}.String())
}

func (os OracleSpec) ToProtoEdge(_ ...any) (*v2.OracleSpecEdge, error) {
	return &v2.OracleSpecEdge{
		Node:   os.ToProto(),
		Cursor: os.Cursor().Encode(),
	}, nil
}
