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
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type ExternalDataSourceSpec struct {
	Spec *DataSourceSpec
}

func (s *ExternalDataSourceSpec) ToProto() *datapb.ExternalDataSourceSpec {
	return &datapb.ExternalDataSourceSpec{
		Spec: s.Spec.ToProto(),
	}
}

func ExternalDataSourceSpecFromProto(spec *datapb.ExternalDataSourceSpec, txHash TxHash, vegaTime time.Time) (*ExternalDataSourceSpec, error) {
	if spec.Spec != nil {
		ds, err := DataSourceSpecFromProto(spec.Spec, txHash, vegaTime)
		if err != nil {
			return nil, err
		}

		return &ExternalDataSourceSpec{
			Spec: ds,
		}, nil
	}

	return &ExternalDataSourceSpec{
		Spec: &DataSourceSpec{},
	}, nil
}

type (
	_Spec  struct{}
	SpecID = ID[_Spec]
)

type (
	Signer  []byte
	Signers = []Signer
)

type DataSourceSpecConfiguration struct {
	Signers Signers
	Filters []Filter
}

type DataSourceSpec struct {
	ID        SpecID
	CreatedAt time.Time
	UpdatedAt time.Time
	Config    *DataSourceSpecConfiguration
	Status    DataSourceSpecStatus
	TxHash    TxHash
	VegaTime  time.Time
}

type DataSourceSpecRaw struct {
	ID        SpecID
	CreatedAt time.Time
	UpdatedAt time.Time
	Signers   Signers
	Filters   []Filter
	Status    DataSourceSpecStatus
	TxHash    TxHash
	VegaTime  time.Time
}

func DataSourceSpecFromProto(spec *datapb.DataSourceSpec, txHash TxHash, vegaTime time.Time) (*DataSourceSpec, error) {
	id := SpecID(spec.Id)
	filters := []Filter{}
	signers := Signers{}

	if spec.Config != nil {
		filters = FiltersFromProto(spec.Config.Filters)
		var err error
		signers, err = SerializeSigners(types.SignersFromProto(spec.Config.Signers))
		if err != nil {
			return nil, err
		}
	}

	return &DataSourceSpec{
		ID:        id,
		CreatedAt: time.Unix(0, spec.CreatedAt),
		UpdatedAt: time.Unix(0, spec.UpdatedAt),
		Config: &DataSourceSpecConfiguration{
			Filters: filters,
			Signers: signers,
		},
		Status:   DataSourceSpecStatus(spec.Status),
		TxHash:   txHash,
		VegaTime: vegaTime,
	}, nil
}

func (ds *DataSourceSpec) ToProto() *datapb.DataSourceSpec {
	filters := []*datapb.Filter{}
	signers := []*datapb.Signer{}

	if ds.Config != nil {
		desSigners := DeserializeSigners(ds.Config.Signers)
		signers = types.SignersIntoProto(desSigners)
		filters = filtersToProto(ds.Config.Filters)
	}

	return &datapb.DataSourceSpec{
		Id:        ds.ID.String(),
		CreatedAt: ds.CreatedAt.UnixNano(),
		UpdatedAt: ds.UpdatedAt.UnixNano(),
		Config: &datapb.DataSourceSpecConfiguration{
			Signers: signers,
			Filters: filters,
		},
		Status: datapb.DataSourceSpec_Status(ds.Status),
	}
}

func (ds *DataSourceSpec) ToOracleProto() *datapb.OracleSpec {
	return &datapb.OracleSpec{
		ExternalDataSourceSpec: &datapb.ExternalDataSourceSpec{
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

func SerializeSigners(signers []*types.Signer) (Signers, error) {
	if len(signers) > 0 {
		sigList := Signers{}

		for _, signer := range signers {
			data, err := signer.Serialize()
			if err != nil {
				return nil, err
			}
			sigList = append(sigList, data)
		}

		return sigList, nil
	}

	return Signers{}, nil
}

func DeserializeSigners(data Signers) []*types.Signer {
	if len(data) > 0 {
		signers := []*types.Signer{}
		for _, s := range data {
			signer := types.DeserializeSigner(s)
			signers = append(signers, signer)
		}

		return signers
	}

	return nil
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
