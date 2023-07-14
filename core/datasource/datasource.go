// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package datasource

import (
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/definition"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type Spec struct {
	ID        string
	CreatedAt int64
	UpdatedAt int64
	Data      *definition.Definition
	Status    common.SpecStatus
}

func (s *Spec) IntoProto() *vegapb.DataSourceSpec {
	config := &vegapb.DataSourceDefinition{}
	if s.Data != nil {
		config = s.Data.IntoProto()
	}

	return &vegapb.DataSourceSpec{
		Id:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Data:      config,
		Status:    s.Status,
	}
}

func (s *Spec) String() string {
	configAsString := ""
	if s.Data != nil {
		configAsString = s.Data.String()
	}

	return fmt.Sprintf(
		"ID(%s) createdAt(%v) updatedAt(%v) data(%s) status(%s)",
		s.ID,
		s.CreatedAt,
		s.UpdatedAt,
		configAsString,
		s.Status.String(),
	)
}

func SpecFromProto(specProto *vegapb.DataSourceSpec) *Spec {
	d, _ := definition.FromProto(specProto.Data)
	//if err != nil {
	// TODO: bubble error
	//}
	return &Spec{
		ID:        specProto.Id,
		CreatedAt: specProto.CreatedAt,
		UpdatedAt: specProto.UpdatedAt,
		Data:      definition.NewWith(d),
		Status:    specProto.Status,
	}
}

func (s *Spec) FromDefinition(d *definition.Definition) *Spec {
	if d != nil {
		bytes, _ := proto.Marshal(d.IntoProto())
		specID := hex.EncodeToString(crypto.Hash(bytes))
		return &Spec{
			ID:   specID,
			Data: d,
		}
	}

	return &Spec{}
}

func SpecFromDefinition(d definition.Definition) *Spec {
	bytes, _ := proto.Marshal(d.IntoProto())
	specID := hex.EncodeToString(crypto.Hash(bytes))
	return &Spec{
		ID:   specID,
		Data: &d,
	}
}

func (s Spec) GetDefinition() definition.Definition {
	if s.Data == nil {
		return definition.Definition{}
	}

	return *s.Data
}

type SpecBindingForFuture struct {
	SettlementDataProperty     string
	TradingTerminationProperty string
}

func (b SpecBindingForFuture) String() string {
	return fmt.Sprintf(
		"settlementData(%s) tradingTermination(%s)",
		b.SettlementDataProperty,
		b.TradingTerminationProperty,
	)
}

func (b SpecBindingForFuture) IntoProto() *vegapb.DataSourceSpecToFutureBinding {
	return &vegapb.DataSourceSpecToFutureBinding{
		SettlementDataProperty:     b.SettlementDataProperty,
		TradingTerminationProperty: b.TradingTerminationProperty,
	}
}

func (b SpecBindingForFuture) DeepClone() *SpecBindingForFuture {
	return &SpecBindingForFuture{
		SettlementDataProperty:     b.SettlementDataProperty,
		TradingTerminationProperty: b.TradingTerminationProperty,
	}
}

func SpecBindingForFutureFromProto(o *vegapb.DataSourceSpecToFutureBinding) *SpecBindingForFuture {
	return &SpecBindingForFuture{
		SettlementDataProperty:     o.SettlementDataProperty,
		TradingTerminationProperty: o.TradingTerminationProperty,
	}
}

func FromOracleSpecProto(specProto *vegapb.OracleSpec) *Spec {
	if specProto.ExternalDataSourceSpec != nil {
		if specProto.ExternalDataSourceSpec.Spec != nil {
			return SpecFromProto(specProto.ExternalDataSourceSpec.Spec)
		}
	}

	return &Spec{}
}

const (
	ContentTypeInvalid                 = definition.ContentTypeInvalid
	ContentTypeOracle                  = definition.ContentTypeOracle
	ContentTypeEthOracle               = definition.ContentTypeEthOracle
	ContentTypeInternalTimeTermination = definition.ContentTypeInternalTimeTermination
)

func NewDefinitionWith(tp common.DataSourceType) *definition.Definition {
	return definition.NewWith(tp)
}

func NewDefinition(tp definition.ContentType) *definition.Definition {
	return definition.New(tp)
}

func DefinitionFromProto(protoConfig *vegapb.DataSourceDefinition) (common.DataSourceType, error) {
	return definition.FromProto(protoConfig)
}
