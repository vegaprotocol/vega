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

package entities

import (
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type OracleSpec struct {
	ExternalDataSourceSpec *ExternalDataSourceSpec
}

func OracleSpecFromProto(spec *vegapb.OracleSpec, txHash TxHash, vegaTime time.Time) *OracleSpec {
	if spec != nil {
		if spec.ExternalDataSourceSpec != nil {
			ds := ExternalDataSourceSpecFromProto(spec.ExternalDataSourceSpec, txHash, vegaTime)

			return &OracleSpec{
				ExternalDataSourceSpec: ds,
			}
		}
	}

	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{},
	}
}

func (os OracleSpec) ToProto() *vegapb.OracleSpec {
	return &vegapb.OracleSpec{
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
