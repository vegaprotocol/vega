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
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// AMMBaseCommand these 3 parameters should be always specified
// in both the the submit and amend commands.
type AMMBaseCommand struct {
	MarketID          string
	Party             string
	SlippageTolerance num.Decimal
}

type ConcentratedLiquidityParameters struct {
	Base                    *num.Uint
	LowerBound              *num.Uint
	UpperBound              *num.Uint
	MarginRatioAtLowerBound *num.Decimal
	MarginRatioAtUpperBound *num.Decimal
}

func (p *ConcentratedLiquidityParameters) ToProtoEvent() *eventspb.AMMPool_ConcentratedLiquidityParameters {
	return &eventspb.AMMPool_ConcentratedLiquidityParameters{
		Base:                    p.Base.String(),
		LowerBound:              p.LowerBound.String(),
		UpperBound:              p.UpperBound.String(),
		MarginRatioAtUpperBound: p.MarginRatioAtUpperBound.String(),
		MarginRatioAtLowerBound: p.MarginRatioAtLowerBound.String(),
	}
}

func (p *ConcentratedLiquidityParameters) ApplyUpdate(update *ConcentratedLiquidityParameters) {
	if update.Base != nil {
		p.Base = update.Base
	}

	if update.LowerBound != nil {
		p.LowerBound = update.LowerBound
	}

	if update.UpperBound != nil {
		p.UpperBound = update.UpperBound
	}

	if update.MarginRatioAtLowerBound != nil {
		p.MarginRatioAtLowerBound = update.MarginRatioAtLowerBound
	}

	if update.MarginRatioAtUpperBound != nil {
		p.MarginRatioAtUpperBound = update.MarginRatioAtUpperBound
	}
}

func (p ConcentratedLiquidityParameters) IntoProto() *commandspb.SubmitAMM_ConcentratedLiquidityParameters {
	ret := &commandspb.SubmitAMM_ConcentratedLiquidityParameters{}
	return ret
}

type SubmitAMM struct {
	AMMBaseCommand
	CommitmentAmount *num.Uint
	Parameters       *ConcentratedLiquidityParameters
}

func NewSubmitAMMFromProto(
	submitAMM *commandspb.SubmitAMM,
	party string,
) *SubmitAMM {
	// all parameters have been validated by the command package here.

	commitment, _ := num.UintFromString(submitAMM.CommitmentAmount, 10)
	base, _ := num.UintFromString(submitAMM.ConcentratedLiquidityParameters.Base, 10)
	lowerBound, _ := num.UintFromString(submitAMM.ConcentratedLiquidityParameters.LowerBound, 10)
	upperBound, _ := num.UintFromString(submitAMM.ConcentratedLiquidityParameters.UpperBound, 10)
	marginRatioAtUpperBound, _ := num.DecimalFromString(submitAMM.ConcentratedLiquidityParameters.MarginRatioAtUpperBound)
	marginRatioAtLowerBound, _ := num.DecimalFromString(submitAMM.ConcentratedLiquidityParameters.MarginRatioAtLowerBound)

	slippage, _ := num.DecimalFromString(submitAMM.SlippageTolerance)

	return &SubmitAMM{
		AMMBaseCommand: AMMBaseCommand{
			Party:             party,
			MarketID:          submitAMM.MarketId,
			SlippageTolerance: slippage,
		},
		CommitmentAmount: commitment,
		Parameters: &ConcentratedLiquidityParameters{
			Base:                    base,
			LowerBound:              lowerBound,
			UpperBound:              upperBound,
			MarginRatioAtLowerBound: &marginRatioAtLowerBound,
			MarginRatioAtUpperBound: &marginRatioAtUpperBound,
		},
	}
}

func (s SubmitAMM) IntoProto() *commandspb.SubmitAMM {
	// set defaults, this is why we don't use a pointer receiver
	zero := num.UintZero() // this call clones, we are just calling String(), so we only need a single 0-value
	if s.CommitmentAmount == nil {
		s.CommitmentAmount = zero
	}
	// create a shallow copy, because this field is a pointer, we mustn't reassign anything
	cpy := *s.Parameters
	s.Parameters = &cpy
	// this should be split to a different function, because this is modifying the
	if cpy.LowerBound == nil {
		cpy.LowerBound = zero
	}
	if cpy.UpperBound == nil {
		cpy.UpperBound = zero
	}
	if cpy.Base == nil {
		cpy.Base = zero
	}
	return &commandspb.SubmitAMM{
		MarketId:          s.MarketID,
		CommitmentAmount:  s.CommitmentAmount.String(),
		SlippageTolerance: s.SlippageTolerance.String(),
		ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
			UpperBound:              s.Parameters.UpperBound.String(),
			LowerBound:              s.Parameters.LowerBound.String(),
			Base:                    s.Parameters.Base.String(),
			MarginRatioAtUpperBound: s.Parameters.MarginRatioAtUpperBound.String(),
			MarginRatioAtLowerBound: s.Parameters.MarginRatioAtLowerBound.String(),
		},
	}
}

type AmendAMM struct {
	AMMBaseCommand
	CommitmentAmount *num.Uint
	Parameters       *ConcentratedLiquidityParameters
}

func (a AmendAMM) IntoProto() *commandspb.AmendAMM {
	ret := &commandspb.AmendAMM{
		MarketId:          a.MarketID,
		CommitmentAmount:  nil,
		SlippageTolerance: a.SlippageTolerance.String(),
	}
	if a.CommitmentAmount != nil {
		ret.CommitmentAmount = ptr.From(a.CommitmentAmount.String())
	}
	if a.Parameters == nil {
		return ret
	}
	ret.ConcentratedLiquidityParameters = &commandspb.AmendAMM_ConcentratedLiquidityParameters{}
	if a.Parameters.Base != nil {
		ret.ConcentratedLiquidityParameters.Base = ptr.From(a.Parameters.Base.String())
	}
	if a.Parameters.LowerBound != nil {
		ret.ConcentratedLiquidityParameters.LowerBound = ptr.From(a.Parameters.LowerBound.String())
	}
	if a.Parameters.UpperBound != nil {
		ret.ConcentratedLiquidityParameters.UpperBound = ptr.From(a.Parameters.UpperBound.String())
	}
	if a.Parameters.MarginRatioAtLowerBound != nil {
		ret.ConcentratedLiquidityParameters.MarginRatioAtLowerBound = ptr.From(a.Parameters.MarginRatioAtLowerBound.String())
	}
	if a.Parameters.MarginRatioAtUpperBound != nil {
		ret.ConcentratedLiquidityParameters.MarginRatioAtUpperBound = ptr.From(a.Parameters.MarginRatioAtUpperBound.String())
	}
	return ret
}

func NewAmendAMMFromProto(
	amendAMM *commandspb.AmendAMM,
	party string,
) *AmendAMM {
	// all parameters have been validated by the command package here.

	var commitment, base, lowerBound, upperBound *num.Uint
	var marginRatioAtUpperBound, marginRatioAtLowerBound *num.Decimal

	// this is optional
	if amendAMM.CommitmentAmount != nil {
		commitment, _ = num.UintFromString(*amendAMM.CommitmentAmount, 10)
	}

	//  this too, and the parameters it contains
	if amendAMM.ConcentratedLiquidityParameters != nil {
		if amendAMM.ConcentratedLiquidityParameters.Base != nil {
			base, _ = num.UintFromString(*amendAMM.ConcentratedLiquidityParameters.Base, 10)
		}
		if amendAMM.ConcentratedLiquidityParameters.LowerBound != nil {
			lowerBound, _ = num.UintFromString(*amendAMM.ConcentratedLiquidityParameters.LowerBound, 10)
		}
		if amendAMM.ConcentratedLiquidityParameters.UpperBound != nil {
			upperBound, _ = num.UintFromString(*amendAMM.ConcentratedLiquidityParameters.UpperBound, 10)
		}
		if amendAMM.ConcentratedLiquidityParameters.MarginRatioAtLowerBound != nil {
			marginRatio, _ := num.DecimalFromString(*amendAMM.ConcentratedLiquidityParameters.MarginRatioAtLowerBound)
			marginRatioAtLowerBound = ptr.From(marginRatio)
		}
		if amendAMM.ConcentratedLiquidityParameters.MarginRatioAtUpperBound != nil {
			marginRatio, _ := num.DecimalFromString(*amendAMM.ConcentratedLiquidityParameters.MarginRatioAtUpperBound)
			marginRatioAtUpperBound = ptr.From(marginRatio)
		}
	}

	slippage, _ := num.DecimalFromString(amendAMM.SlippageTolerance)

	return &AmendAMM{
		AMMBaseCommand: AMMBaseCommand{
			Party:             party,
			MarketID:          amendAMM.MarketId,
			SlippageTolerance: slippage,
		},
		CommitmentAmount: commitment,
		Parameters: &ConcentratedLiquidityParameters{
			Base:                    base,
			LowerBound:              lowerBound,
			UpperBound:              upperBound,
			MarginRatioAtUpperBound: marginRatioAtUpperBound,
			MarginRatioAtLowerBound: marginRatioAtLowerBound,
		},
	}
}

type CancelAMM struct {
	MarketID string
	Party    string
	Method   AMMPoolCancellationMethod
}

func (c CancelAMM) IntoProto() *commandspb.CancelAMM {
	return &commandspb.CancelAMM{
		MarketId: c.MarketID,
		Method:   c.Method,
	}
}

func NewCancelAMMFromProto(
	cancelAMM *commandspb.CancelAMM,
	party string,
) *CancelAMM {
	return &CancelAMM{
		MarketID: cancelAMM.MarketId,
		Party:    party,
		Method:   cancelAMM.Method,
	}
}

type AMMPoolCancellationMethod = commandspb.CancelAMM_Method

const (
	AMMPoolCancellationMethodUnspecified AMMPoolCancellationMethod = commandspb.CancelAMM_METHOD_UNSPECIFIED
	AMMPoolCancellationMethodImmediate                             = commandspb.CancelAMM_METHOD_IMMEDIATE
	AMMPoolCancellationMethodReduceOnly                            = commandspb.CancelAMM_METHOD_REDUCE_ONLY
)

type AMMPoolStatusReason = eventspb.AMMPool_StatusReason

const (
	AMMPoolStatusReasonUnspecified           AMMPoolStatusReason = eventspb.AMMPool_STATUS_REASON_UNSPECIFIED
	AMMPoolStatusReasonCancelledByParty                          = eventspb.AMMPool_STATUS_REASON_CANCELLED_BY_PARTY
	AMMPoolStatusReasonCannotFillCommitment                      = eventspb.AMMPool_STATUS_REASON_CANNOT_FILL_COMMITMENT
	AMMPoolStatusReasonPartyAlreadyOwnsAPool                     = eventspb.AMMPool_STATUS_REASON_PARTY_ALREADY_OWNS_A_POOL
	AMMPoolStatusReasonPartyClosedOut                            = eventspb.AMMPool_STATUS_REASON_PARTY_CLOSED_OUT
	AMMPoolStatusReasonMarketClosed                              = eventspb.AMMPool_STATUS_REASON_MARKET_CLOSED
	AMMPoolStatusReasonCommitmentTooLow                          = eventspb.AMMPool_STATUS_REASON_COMMITMENT_TOO_LOW
	AMMPoolStatusReasonCannotRebase                              = eventspb.AMMPool_STATUS_REASON_CANNOT_REBASE
)

type AMMPoolStatus = eventspb.AMMPool_Status

const (
	AMMPoolStatusUnspecified AMMPoolStatus = eventspb.AMMPool_STATUS_UNSPECIFIED
	AMMPoolStatusActive                    = eventspb.AMMPool_STATUS_ACTIVE
	AMMPoolStatusRejected                  = eventspb.AMMPool_STATUS_REJECTED
	AMMPoolStatusCancelled                 = eventspb.AMMPool_STATUS_CANCELLED
	AMMPoolStatusStopped                   = eventspb.AMMPool_STATUS_STOPPED
	AMMPoolStatusReduceOnly                = eventspb.AMMPool_STATUS_REDUCE_ONLY
)
