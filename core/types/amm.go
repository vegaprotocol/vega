package types

import (
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// AMMBaseCommand these 3 parameters should be always specified
// in both the the submit and amend commands
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

type AmendAMM struct {
	AMMBaseCommand
	CommitmentAmount *num.Uint
	Parameters       *ConcentratedLiquidityParameters
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
}

func NewCancelAMMFromProto(
	cancelAMM *commandspb.CancelAMM,
	party string,
) *CancelAMM {
	return &CancelAMM{
		MarketID: cancelAMM.MarketId,
		Party:    party,
	}
}

type AMMPoolStatusReason = eventspb.AMMPool_StatusReason

const (
	AMMPoolStatusReasonUnspecified          AMMPoolStatusReason = eventspb.AMMPool_STATUS_REASON_UNSPECIFIED
	AMMPoolStatusReasonCancelledByParty                         = eventspb.AMMPool_STATUS_REASON_CANCELLED_BY_PARTY
	AMMPoolStatusReasonCannotFillCommitment                     = eventspb.AMMPool_STATUS_REASON_CANNOT_FILL_COMMITMENT
	AMMPoolStatusReasonPartyAlreadyOwnAPool                     = eventspb.AMMPool_STATUS_REASON_PARTY_ALREADY_OWN_A_POOL
	AMMPoolStatusReasonPartyClosedOut                           = eventspb.AMMPool_STATUS_REASON_PARTY_CLOSED_OUT
	AMMPoolStatusReasonMarketClosed                             = eventspb.AMMPool_STATUS_REASON_MARKET_CLOSED
	AMMPoolStatusReasonCommitmentTooLow                         = eventspb.AMMPool_STATUS_REASON_COMMITMENT_TOO_LOW
)

type AMMPoolStatus = eventspb.AMMPool_Status

const (
	AMMPoolStatusUnspecified AMMPoolStatus = eventspb.AMMPool_STATUS_UNSPECIFIED
	AMMPoolStatusActive                    = eventspb.AMMPool_STATUS_ACTIVE
	AMMPoolStatusRejected                  = eventspb.AMMPool_STATUS_REJECTED
	AMMPoolStatusCancelled                 = eventspb.AMMPool_STATUS_CANCELLED
	AMMPoolStatusStopped                   = eventspb.AMMPool_STATUS_STOPPED
)
