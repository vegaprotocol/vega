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
	d, _ := definition.FromProto(specProto.Data, nil)
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

type SpecBindingForCompositePrice struct {
	PriceSourceProperty string
}

func (b SpecBindingForCompositePrice) String() string {
	return fmt.Sprintf(
		"priceSource(%s)",
		b.PriceSourceProperty,
	)
}

func (b SpecBindingForCompositePrice) IntoProto() *vegapb.SpecBindingForCompositePrice {
	return &vegapb.SpecBindingForCompositePrice{
		PriceSourceProperty: b.PriceSourceProperty,
	}
}

func (b SpecBindingForCompositePrice) DeepClone() *SpecBindingForCompositePrice {
	return &SpecBindingForCompositePrice{
		PriceSourceProperty: b.PriceSourceProperty,
	}
}

func SpecBindingForCompositePriceFromProto(o *vegapb.SpecBindingForCompositePrice) *SpecBindingForCompositePrice {
	return &SpecBindingForCompositePrice{
		PriceSourceProperty: o.PriceSourceProperty,
	}
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

type SpecBindingForAutomatedPurchase struct {
	AuctionScheduleProperty       string
	AuctionVolumeSnapshotProperty string
}

func (b SpecBindingForAutomatedPurchase) String() string {
	return fmt.Sprintf(
		"auctionScheduleProperty(%s) auctionVolumeSnapshotProperty(%s)",
		b.AuctionScheduleProperty,
		b.AuctionVolumeSnapshotProperty,
	)
}

func (b SpecBindingForAutomatedPurchase) IntoProto() *vegapb.DataSourceSpecToAutomatedPurchaseBinding {
	return &vegapb.DataSourceSpecToAutomatedPurchaseBinding{
		AuctionScheduleProperty:               b.AuctionScheduleProperty,
		AuctionVolumeSnapshotScheduleProperty: b.AuctionVolumeSnapshotProperty,
	}
}

func (b SpecBindingForAutomatedPurchase) DeepClone() *SpecBindingForAutomatedPurchase {
	return &SpecBindingForAutomatedPurchase{
		AuctionScheduleProperty:       b.AuctionScheduleProperty,
		AuctionVolumeSnapshotProperty: b.AuctionVolumeSnapshotProperty,
	}
}

func SpecBindingForAutomatedPurchaseFromProto(o *vegapb.DataSourceSpecToAutomatedPurchaseBinding) *SpecBindingForAutomatedPurchase {
	return &SpecBindingForAutomatedPurchase{
		AuctionScheduleProperty:       o.AuctionScheduleProperty,
		AuctionVolumeSnapshotProperty: o.AuctionVolumeSnapshotScheduleProperty,
	}
}

type SpecBindingForPerps struct {
	SettlementDataProperty     string
	SettlementScheduleProperty string
}

func (b SpecBindingForPerps) String() string {
	return fmt.Sprintf(
		"settlementData(%s) settlementSchedule(%s)",
		b.SettlementDataProperty,
		b.SettlementScheduleProperty,
	)
}

func (b SpecBindingForPerps) IntoProto() *vegapb.DataSourceSpecToPerpetualBinding {
	return &vegapb.DataSourceSpecToPerpetualBinding{
		SettlementDataProperty:     b.SettlementDataProperty,
		SettlementScheduleProperty: b.SettlementScheduleProperty,
	}
}

func (b SpecBindingForPerps) DeepClone() *SpecBindingForPerps {
	return &SpecBindingForPerps{
		SettlementDataProperty:     b.SettlementDataProperty,
		SettlementScheduleProperty: b.SettlementScheduleProperty,
	}
}

func SpecBindingForPerpsFromProto(o *vegapb.DataSourceSpecToPerpetualBinding) *SpecBindingForPerps {
	return &SpecBindingForPerps{
		SettlementDataProperty:     o.SettlementDataProperty,
		SettlementScheduleProperty: o.SettlementScheduleProperty,
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
	ContentTypeInvalid                        = definition.ContentTypeInvalid
	ContentTypeOracle                         = definition.ContentTypeOracle
	ContentTypeEthOracle                      = definition.ContentTypeEthOracle
	ContentTypeInternalTimeTermination        = definition.ContentTypeInternalTimeTermination
	ContentTypeInternalTimeTriggerTermination = definition.ContentTypeInternalTimeTriggerTermination
)

func NewDefinitionWith(tp common.DataSourceType) *definition.Definition {
	return definition.NewWith(tp)
}

func NewDefinition(tp definition.ContentType) *definition.Definition {
	return definition.New(tp)
}

func DefinitionFromProto(protoConfig *vegapb.DataSourceDefinition) (common.DataSourceType, error) {
	return definition.FromProto(protoConfig, nil)
}
