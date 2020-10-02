package netparams

func defaultNetParams() map[string]value {
	return map[string]value{
		MarketMarginScalingFactorSearchLevel:       NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1.1"),
		MarketMarginScalingFactorInitialMargin:     NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1.2"),
		MarketMarginScalingFactorCollateralRelease: NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1.4"),
		MarketFeeFactorsMakerFee:                   NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00025"),
		MarketFeeFactorsInfrastructureFee:          NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.0005"),
		MarketFeeFactorsLiquidityFee:               NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.001"),

		// governance market proposal
		GovernanceProposalMarketRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalMarketRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalMarketMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalMarketMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),

		// governance asset proposal
		GovernanceProposalAssetRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalAssetMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),

		// governance update market proposal
		GovernanceProposalUpdateMarketRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateMarketMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),

		// governance UpdateNetParam proposal
		GovernanceProposalUpdateNetParamRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateNetParamMinProposerBalance:    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamMinVoterBalance:       NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
	}

}
