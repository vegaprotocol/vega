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

package types

import (
	"fmt"
	"time"

	dsdefinition "code.vegaprotocol.io/vega/core/datasource/definition"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsNewProtocolAutomatedPurchase struct {
	NewProtocolAutomatedPurchase *NewProtocolAutomatedPurchase
}

func (a ProposalTermsNewProtocolAutomatedPurchase) String() string {
	return fmt.Sprintf(
		"NewProtocolAutomatedPurchaseConfiguration(%s)",
		stringer.PtrToString(a.NewProtocolAutomatedPurchase),
	)
}

func (a ProposalTermsNewProtocolAutomatedPurchase) isPTerm() {}

func (a ProposalTermsNewProtocolAutomatedPurchase) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_NewProtocolAutomatedPurchase{
		NewProtocolAutomatedPurchase: a.NewProtocolAutomatedPurchase.IntoProto(),
	}
}

func (a ProposalTermsNewProtocolAutomatedPurchase) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_NewProtocolAutomatedPurchase{
		NewProtocolAutomatedPurchase: a.NewProtocolAutomatedPurchase.IntoProto(),
	}
}

func (a ProposalTermsNewProtocolAutomatedPurchase) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewProtocolAutomatedPurchase
}

func (a ProposalTermsNewProtocolAutomatedPurchase) DeepClone() ProposalTerm {
	if a.NewProtocolAutomatedPurchase == nil {
		return &ProposalTermsNewProtocolAutomatedPurchase{}
	}
	return &ProposalTermsNewProtocolAutomatedPurchase{
		NewProtocolAutomatedPurchase: a.NewProtocolAutomatedPurchase.DeepClone(),
	}
}

func NewProtocolAutomatedPurchaseConfigurationProposalFromProto(
	NewProtocolAutomatedPurchaseProto *vegapb.NewProtocolAutomatedPurchase,
) (*ProposalTermsNewProtocolAutomatedPurchase, error) {
	return &ProposalTermsNewProtocolAutomatedPurchase{
		NewProtocolAutomatedPurchase: NewProtocolAutomatedPurchaseFromProto(NewProtocolAutomatedPurchaseProto),
	}, nil
}

type NewProtocolAutomatedPurchase struct {
	Changes *NewProtocolAutomatedPurchaseChanges
}

func (p NewProtocolAutomatedPurchase) IntoProto() *vegapb.NewProtocolAutomatedPurchase {
	return &vegapb.NewProtocolAutomatedPurchase{
		Changes: p.Changes.IntoProto(),
	}
}

func (p NewProtocolAutomatedPurchase) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.PtrToString(p.Changes),
	)
}

func (p NewProtocolAutomatedPurchase) DeepClone() *NewProtocolAutomatedPurchase {
	if p.Changes == nil {
		return &NewProtocolAutomatedPurchase{}
	}
	return &NewProtocolAutomatedPurchase{
		Changes: p.Changes.DeepClone(),
	}
}

func NewProtocolAutomatedPurchaseFromProto(p *vegapb.NewProtocolAutomatedPurchase) *NewProtocolAutomatedPurchase {
	if p == nil {
		return &NewProtocolAutomatedPurchase{}
	}
	return &NewProtocolAutomatedPurchase{
		Changes: NewProtocolAutomatedPurchaseChangesFromProto(p.Changes),
	}
}

type NewProtocolAutomatedPurchaseChanges struct {
	From                          string
	FromAccountType               AccountType
	ToAccountType                 AccountType
	MarketID                      string
	PriceOracle                   *vegapb.DataSourceDefinition
	PriceOracleBinding            *vegapb.SpecBindingForCompositePrice
	OracleOffsetFactor            num.Decimal
	AuctionSchedule               *vegapb.DataSourceDefinition
	AuctionVolumeSnapshotSchedule *vegapb.DataSourceDefinition
	AutomatedPurchaseSpecBinding  *vegapb.DataSourceSpecToAutomatedPurchaseBinding
	AuctionDuration               time.Duration
	MinimumAuctionSize            *num.Uint
	MaximumAuctionSize            *num.Uint
	ExpiryTimestamp               time.Time
	OraclePriceStalenessTolerance time.Duration
}

func (apc *NewProtocolAutomatedPurchaseChanges) DeepClone() *NewProtocolAutomatedPurchaseChanges {
	cloned := &NewProtocolAutomatedPurchaseChanges{
		From:                          apc.From,
		FromAccountType:               apc.FromAccountType,
		ToAccountType:                 apc.ToAccountType,
		MarketID:                      apc.MarketID,
		AuctionDuration:               apc.AuctionDuration,
		MinimumAuctionSize:            apc.MinimumAuctionSize,
		MaximumAuctionSize:            apc.MaximumAuctionSize,
		OracleOffsetFactor:            apc.OracleOffsetFactor,
		PriceOracleBinding:            apc.PriceOracleBinding,
		AutomatedPurchaseSpecBinding:  apc.AutomatedPurchaseSpecBinding,
		ExpiryTimestamp:               apc.ExpiryTimestamp,
		OraclePriceStalenessTolerance: apc.OraclePriceStalenessTolerance,
	}

	asDefinition, _ := dsdefinition.FromProto(apc.AuctionSchedule, nil)
	cloned.AuctionSchedule, _ = asDefinition.DeepClone().ToDefinitionProto()
	opDefinition, _ := dsdefinition.FromProto(apc.PriceOracle, nil)
	cloned.PriceOracle, _ = opDefinition.DeepClone().ToDefinitionProto()
	avssoDefinition, _ := dsdefinition.FromProto(apc.AuctionVolumeSnapshotSchedule, nil)
	cloned.AuctionVolumeSnapshotSchedule, _ = avssoDefinition.DeepClone().ToDefinitionProto()

	return cloned
}

func (apc *NewProtocolAutomatedPurchaseChanges) IntoProto() *vegapb.NewProtocolAutomatedPurchaseChanges {
	return &vegapb.NewProtocolAutomatedPurchaseChanges{
		From:                          apc.From,
		FromAccountType:               apc.FromAccountType,
		ToAccountType:                 apc.ToAccountType,
		MarketId:                      apc.MarketID,
		PriceOracle:                   apc.PriceOracle,
		PriceOracleSpecBinding:        apc.PriceOracleBinding,
		OracleOffsetFactor:            apc.OracleOffsetFactor.String(),
		AuctionSchedule:               apc.AuctionSchedule,
		AuctionVolumeSnapshotSchedule: apc.AuctionVolumeSnapshotSchedule,
		AutomatedPurchaseSpecBinding:  apc.AutomatedPurchaseSpecBinding,
		AuctionDuration:               apc.AuctionDuration.String(),
		MinimumAuctionSize:            apc.MinimumAuctionSize.String(),
		MaximumAuctionSize:            apc.MaximumAuctionSize.String(),
		ExpiryTimestamp:               apc.ExpiryTimestamp.Unix(),
		OraclePriceStalenessTolerance: apc.OraclePriceStalenessTolerance.String(),
	}
}

func NewProtocolAutomatedPurchaseChangesFromProto(p *vegapb.NewProtocolAutomatedPurchaseChanges) *NewProtocolAutomatedPurchaseChanges {
	auctionDuration, _ := time.ParseDuration(p.AuctionDuration)
	minSize, _ := num.UintFromString(p.MinimumAuctionSize, 10)
	maxSize, _ := num.UintFromString(p.MaximumAuctionSize, 10)
	oraclePriceStalenessTolerance, _ := time.ParseDuration(p.OraclePriceStalenessTolerance)
	return &NewProtocolAutomatedPurchaseChanges{
		From:                          p.From,
		FromAccountType:               p.FromAccountType,
		ToAccountType:                 p.ToAccountType,
		MarketID:                      p.MarketId,
		PriceOracle:                   p.PriceOracle,
		PriceOracleBinding:            p.PriceOracleSpecBinding,
		OracleOffsetFactor:            num.MustDecimalFromString(p.OracleOffsetFactor),
		AuctionSchedule:               p.AuctionSchedule,
		AuctionDuration:               auctionDuration,
		AuctionVolumeSnapshotSchedule: p.AuctionVolumeSnapshotSchedule,
		AutomatedPurchaseSpecBinding:  p.AutomatedPurchaseSpecBinding,
		MinimumAuctionSize:            minSize,
		MaximumAuctionSize:            maxSize,
		ExpiryTimestamp:               time.Unix(p.ExpiryTimestamp, 0),
		OraclePriceStalenessTolerance: oraclePriceStalenessTolerance,
	}
}
