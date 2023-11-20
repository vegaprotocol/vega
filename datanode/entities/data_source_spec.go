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
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type DataSourceSpec struct {
	ID        SpecID
	CreatedAt time.Time
	UpdatedAt time.Time
	Data      *DataSourceDefinition
	Status    DataSourceSpecStatus
	TxHash    TxHash
	VegaTime  time.Time
}

func DataSourceSpecFromProto(protoSpec *vegapb.DataSourceSpec, txHash TxHash, vegaTime time.Time) *DataSourceSpec {
	spec := &DataSourceSpec{}

	if protoSpec != nil {
		spec.ID = SpecID(protoSpec.Id)
		spec.CreatedAt = time.Unix(0, protoSpec.CreatedAt)
		spec.UpdatedAt = time.Unix(0, protoSpec.UpdatedAt)
		spec.Data = ptr.From(DataSourceDefinition{})
		spec.Status = DataSourceSpecStatus(protoSpec.Status)
		spec.TxHash = txHash
		spec.VegaTime = vegaTime

		if protoSpec.Data != nil && protoSpec.Data.SourceType != nil {
			spec.Data = &DataSourceDefinition{protoSpec.Data}
		}
	}

	return spec
}

func (ds *DataSourceSpec) ToProto() *vegapb.DataSourceSpec {
	protoData := &vegapb.DataSourceSpec{}

	if ds != nil {
		if ds.Data != nil && *ds.Data != (DataSourceDefinition{}) {
			protoData.Id = ds.ID.String()
			protoData.CreatedAt = ds.CreatedAt.UnixNano()
			protoData.UpdatedAt = ds.UpdatedAt.UnixNano()
			protoData.Data = &vegapb.DataSourceDefinition{}
			protoData.Status = vegapb.DataSourceSpec_Status(ds.Status)
			if ds.Data.SourceType != nil {
				protoData.Data = ds.Data.DataSourceDefinition
			}
		}
	}

	return protoData
}

func (ds *DataSourceSpec) ToOracleProto() *vegapb.OracleSpec {
	return &vegapb.OracleSpec{
		ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
			Spec: ds.ToProto(),
		},
	}
}

func (ds DataSourceSpec) Cursor() *Cursor {
	return NewCursor(DataSourceSpecCursor{ds.VegaTime, ds.ID}.String())
}

func (ds DataSourceSpec) ToOracleProtoEdge(_ ...any) (*v2.OracleSpecEdge, error) {
	return &v2.OracleSpecEdge{
		Node:   ds.ToOracleProto(),
		Cursor: ds.Cursor().Encode(),
	}, nil
}

type ExternalDataSourceSpec struct {
	Spec *DataSourceSpec
}

func (s *ExternalDataSourceSpec) ToProto() *vegapb.ExternalDataSourceSpec {
	return &vegapb.ExternalDataSourceSpec{
		Spec: s.Spec.ToProto(),
	}
}

func ExternalDataSourceSpecFromProto(spec *vegapb.ExternalDataSourceSpec, txHash TxHash, vegaTime time.Time) *ExternalDataSourceSpec {
	if spec != nil {
		if spec.Spec != nil {
			return &ExternalDataSourceSpec{
				Spec: DataSourceSpecFromProto(spec.Spec, txHash, vegaTime),
			}
		}
	}

	return &ExternalDataSourceSpec{
		Spec: &DataSourceSpec{
			Data: ptr.From(DataSourceDefinition{}),
		},
	}
}

type DataSourceSpecCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	ID       SpecID    `json:"id"`
}

func (ds DataSourceSpecCursor) String() string {
	bs, err := json.Marshal(ds)
	if err != nil {
		panic(fmt.Errorf("could not marshal oracle spec cursor: %w", err))
	}
	return string(bs)
}

func (ds *DataSourceSpecCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), ds)
}
