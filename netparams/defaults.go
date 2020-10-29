package netparams

import "time"

func defaultNetParams() map[string]value {
	return map[string]value{
		// markets
		MarketMarginScalingFactorSearchLevel:          NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1.1"),
		MarketMarginScalingFactorInitialMargin:        NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1.2"),
		MarketMarginScalingFactorCollateralRelease:    NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1.4"),
		MarketFeeFactorsMakerFee:                      NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00025"),
		MarketFeeFactorsInfrastructureFee:             NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.0005"),
		MarketFeeFactorsLiquidityFee:                  NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.001"),
		MarketAuctionMinimumDuration:                  NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("30m0s"),
		MarketAuctionMaximumDuration:                  NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("168h0m0s"),
		MarketInitialMarkPrice:                        NewInt(IntGT(0)).Mutable(true).MustUpdate("1"),
		MarketLiquidityBondPenaltyParameter:           NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1"),
		MarketLiquidityMaximumLiquidityFeeFactorLevel: NewFloat(FloatGT(0), FloatLTE(1)).Mutable(true).MustUpdate("1"),
		MarketLiquidityStakeToCCYSiskas:               NewFloat(FloatGT(0)).Mutable(true).MustUpdate("1"),

		// governance market proposal
		GovernanceProposalMarketMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalMarketRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalMarketMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalMarketMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),

		// governance asset proposal
		GovernanceProposalAssetMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalAssetMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),

		// governance update market proposal
		GovernanceProposalUpdateMarketMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateMarketMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),

		// governance UpdateNetParam proposal
		GovernanceProposalUpdateNetParamMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateNetParamMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
	}
}
