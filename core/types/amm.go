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

type OrderbookShapeResult struct {
	AmmParty string
	Buys     []*Order
	Sells    []*Order
	Approx   bool
}

// AMMBaseCommand these 3 parameters should be always specified
// in both the the submit and amend commands.
type AMMBaseCommand struct {
	MarketID          string
	Party             string
	SlippageTolerance num.Decimal
	ProposedFee       num.Decimal
}

type ConcentratedLiquidityParameters struct {
	Base                 *num.Uint
	LowerBound           *num.Uint
	UpperBound           *num.Uint
	LeverageAtLowerBound *num.Decimal
	LeverageAtUpperBound *num.Decimal
}

func (p *ConcentratedLiquidityParameters) ToProtoEvent() *eventspb.AMM_ConcentratedLiquidityParameters {
	var upper, lower *string
	if p.UpperBound != nil {
		upper = ptr.From(p.UpperBound.String())
	}

	if p.LowerBound != nil {
		lower = ptr.From(p.LowerBound.String())
	}

	var lowerLeverage, upperLeverage *string
	if p.LeverageAtLowerBound != nil {
		lowerLeverage = ptr.From(p.LeverageAtLowerBound.String())
	}

	if p.LeverageAtUpperBound != nil {
		upperLeverage = ptr.From(p.LeverageAtUpperBound.String())
	}
	return &eventspb.AMM_ConcentratedLiquidityParameters{
		Base:                 p.Base.String(),
		LowerBound:           lower,
		UpperBound:           upper,
		LeverageAtUpperBound: upperLeverage,
		LeverageAtLowerBound: lowerLeverage,
	}
}

func (p ConcentratedLiquidityParameters) Clone() *ConcentratedLiquidityParameters {
	ret := &ConcentratedLiquidityParameters{}
	if p.Base != nil {
		ret.Base = p.Base.Clone()
	}
	if p.LowerBound != nil {
		ret.LowerBound = p.LowerBound.Clone()
	}
	if p.UpperBound != nil {
		ret.UpperBound = p.UpperBound.Clone()
	}
	if p.LeverageAtLowerBound != nil {
		cpy := *p.LeverageAtLowerBound
		ret.LeverageAtLowerBound = &cpy
	}
	if p.LeverageAtUpperBound != nil {
		cpy := *p.LeverageAtUpperBound
		ret.LeverageAtUpperBound = &cpy
	}
	return ret
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
	var (
		upperBound, lowerBound       *num.Uint
		upperLeverage, lowerLeverage *num.Decimal
	)

	commitment, _ := num.UintFromString(submitAMM.CommitmentAmount, 10)

	params := submitAMM.ConcentratedLiquidityParameters
	base, _ := num.UintFromString(params.Base, 10)
	if params.LowerBound != nil {
		lowerBound, _ = num.UintFromString(*params.LowerBound, 10)
	}

	if params.LeverageAtLowerBound != nil {
		leverage, _ := num.DecimalFromString(*params.LeverageAtLowerBound)
		lowerLeverage = ptr.From(leverage)
	}

	if params.UpperBound != nil {
		upperBound, _ = num.UintFromString(*params.UpperBound, 10)
	}

	if params.LeverageAtUpperBound != nil {
		leverage, _ := num.DecimalFromString(*params.LeverageAtUpperBound)
		upperLeverage = ptr.From(leverage)
	}

	slippage, _ := num.DecimalFromString(submitAMM.SlippageTolerance)
	proposedFee, _ := num.DecimalFromString(submitAMM.ProposedFee)
	return &SubmitAMM{
		AMMBaseCommand: AMMBaseCommand{
			Party:             party,
			MarketID:          submitAMM.MarketId,
			SlippageTolerance: slippage,
			ProposedFee:       proposedFee,
		},
		CommitmentAmount: commitment,
		Parameters: &ConcentratedLiquidityParameters{
			Base:                 base,
			LowerBound:           lowerBound,
			UpperBound:           upperBound,
			LeverageAtLowerBound: lowerLeverage,
			LeverageAtUpperBound: upperLeverage,
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
	var lower, upper, leverageLower, leverageUpper *string
	var base string
	if s.Parameters.LowerBound != nil {
		lower = ptr.From(s.Parameters.LowerBound.String())
	}

	if s.Parameters.LeverageAtLowerBound != nil {
		leverageLower = ptr.From(s.Parameters.LeverageAtLowerBound.String())
	}

	if s.Parameters.UpperBound != nil {
		upper = ptr.From(s.Parameters.UpperBound.String())
	}

	if s.Parameters.LeverageAtUpperBound != nil {
		leverageUpper = ptr.From(s.Parameters.LeverageAtUpperBound.String())
	}

	if s.Parameters.Base != nil {
		base = s.Parameters.Base.String()
	}
	return &commandspb.SubmitAMM{
		MarketId:          s.MarketID,
		CommitmentAmount:  s.CommitmentAmount.String(),
		SlippageTolerance: s.SlippageTolerance.String(),
		ProposedFee:       s.ProposedFee.String(),
		ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
			UpperBound:           upper,
			LowerBound:           lower,
			Base:                 base,
			LeverageAtUpperBound: leverageUpper,
			LeverageAtLowerBound: leverageLower,
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
		ProposedFee:       nil,
	}
	if a.CommitmentAmount != nil {
		ret.CommitmentAmount = ptr.From(a.CommitmentAmount.String())
	}
	if !a.ProposedFee.IsZero() {
		ret.ProposedFee = ptr.From(a.ProposedFee.String())
	}
	if a.Parameters == nil {
		return ret
	}
	ret.ConcentratedLiquidityParameters = &commandspb.AmendAMM_ConcentratedLiquidityParameters{}
	if a.Parameters.Base != nil {
		ret.ConcentratedLiquidityParameters.Base = a.Parameters.Base.String()
	}
	if a.Parameters.LowerBound != nil {
		ret.ConcentratedLiquidityParameters.LowerBound = ptr.From(a.Parameters.LowerBound.String())
	}
	if a.Parameters.UpperBound != nil {
		ret.ConcentratedLiquidityParameters.UpperBound = ptr.From(a.Parameters.UpperBound.String())
	}
	if a.Parameters.LeverageAtLowerBound != nil {
		ret.ConcentratedLiquidityParameters.LeverageAtLowerBound = ptr.From(a.Parameters.LeverageAtLowerBound.String())
	}
	if a.Parameters.LeverageAtUpperBound != nil {
		ret.ConcentratedLiquidityParameters.LeverageAtUpperBound = ptr.From(a.Parameters.LeverageAtUpperBound.String())
	}
	return ret
}

func NewAmendAMMFromProto(
	amendAMM *commandspb.AmendAMM,
	party string,
) *AmendAMM {
	// all parameters have been validated by the command package here.

	var commitment, base, lowerBound, upperBound *num.Uint
	var leverageAtUpperBound, leverageAtLowerBound *num.Decimal

	// this is optional
	if amendAMM.CommitmentAmount != nil {
		commitment, _ = num.UintFromString(*amendAMM.CommitmentAmount, 10)
	}

	//  this too, and the parameters it contains
	if amendAMM.ConcentratedLiquidityParameters != nil {
		base, _ = num.UintFromString(amendAMM.ConcentratedLiquidityParameters.Base, 10)
		if amendAMM.ConcentratedLiquidityParameters.LowerBound != nil {
			lowerBound, _ = num.UintFromString(*amendAMM.ConcentratedLiquidityParameters.LowerBound, 10)
		}
		if amendAMM.ConcentratedLiquidityParameters.UpperBound != nil {
			upperBound, _ = num.UintFromString(*amendAMM.ConcentratedLiquidityParameters.UpperBound, 10)
		}
		if amendAMM.ConcentratedLiquidityParameters.LeverageAtLowerBound != nil {
			leverage, _ := num.DecimalFromString(*amendAMM.ConcentratedLiquidityParameters.LeverageAtLowerBound)
			leverageAtLowerBound = ptr.From(leverage)
		}
		if amendAMM.ConcentratedLiquidityParameters.LeverageAtUpperBound != nil {
			leverage, _ := num.DecimalFromString(*amendAMM.ConcentratedLiquidityParameters.LeverageAtUpperBound)
			leverageAtUpperBound = ptr.From(leverage)
		}
	}

	slippage, _ := num.DecimalFromString(amendAMM.SlippageTolerance)

	var proposedFee num.Decimal
	if amendAMM.ProposedFee != nil {
		proposedFee, _ = num.DecimalFromString(*amendAMM.ProposedFee)
	}

	return &AmendAMM{
		AMMBaseCommand: AMMBaseCommand{
			Party:             party,
			MarketID:          amendAMM.MarketId,
			SlippageTolerance: slippage,
			ProposedFee:       proposedFee,
		},
		CommitmentAmount: commitment,
		Parameters: &ConcentratedLiquidityParameters{
			Base:                 base,
			LowerBound:           lowerBound,
			UpperBound:           upperBound,
			LeverageAtUpperBound: leverageAtUpperBound,
			LeverageAtLowerBound: leverageAtLowerBound,
		},
	}
}

type CancelAMM struct {
	MarketID string
	Party    string
	Method   AMMCancellationMethod
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

type AMMCancellationMethod = commandspb.CancelAMM_Method

const (
	AMMCancellationMethodUnspecified AMMCancellationMethod = commandspb.CancelAMM_METHOD_UNSPECIFIED
	AMMCancellationMethodImmediate                         = commandspb.CancelAMM_METHOD_IMMEDIATE
	AMMCancellationMethodReduceOnly                        = commandspb.CancelAMM_METHOD_REDUCE_ONLY
)

type AMMStatusReason = eventspb.AMM_StatusReason

const (
	AMMStatusReasonUnspecified           AMMStatusReason = eventspb.AMM_STATUS_REASON_UNSPECIFIED
	AMMStatusReasonCancelledByParty                      = eventspb.AMM_STATUS_REASON_CANCELLED_BY_PARTY
	AMMStatusReasonCannotFillCommitment                  = eventspb.AMM_STATUS_REASON_CANNOT_FILL_COMMITMENT
	AMMStatusReasonPartyAlreadyOwnsAPool                 = eventspb.AMM_STATUS_REASON_PARTY_ALREADY_OWNS_AMM_FOR_MARKET
	AMMStatusReasonPartyClosedOut                        = eventspb.AMM_STATUS_REASON_PARTY_CLOSED_OUT
	AMMStatusReasonMarketClosed                          = eventspb.AMM_STATUS_REASON_MARKET_CLOSED
	AMMStatusReasonCommitmentTooLow                      = eventspb.AMM_STATUS_REASON_COMMITMENT_TOO_LOW
	AMMStatusReasonCannotRebase                          = eventspb.AMM_STATUS_REASON_CANNOT_REBASE
)

type AMMPoolStatus = eventspb.AMM_Status

const (
	AMMPoolStatusUnspecified AMMPoolStatus = eventspb.AMM_STATUS_UNSPECIFIED
	AMMPoolStatusActive                    = eventspb.AMM_STATUS_ACTIVE
	AMMPoolStatusRejected                  = eventspb.AMM_STATUS_REJECTED
	AMMPoolStatusCancelled                 = eventspb.AMM_STATUS_CANCELLED
	AMMPoolStatusStopped                   = eventspb.AMM_STATUS_STOPPED
	AMMPoolStatusReduceOnly                = eventspb.AMM_STATUS_REDUCE_ONLY
)
