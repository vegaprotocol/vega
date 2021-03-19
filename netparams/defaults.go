package netparams

import (
	"time"

	"code.vegaprotocol.io/vega/netparams/checks"
	"code.vegaprotocol.io/vega/proto"
)

const (
	week = "168h0m0s"
)

func defaultNetParams() map[string]value {
	return map[string]value{
		// markets
		MarketMarginScalingFactors:                      NewJSON(&proto.ScalingFactors{}, checks.MarginScalingFactor()).Mutable(true).MustUpdate(`{"search_level": 1.1, "initial_margin": 1.2, "collateral_release": 1.4}`),
		MarketFeeFactorsMakerFee:                        NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00025"),
		MarketFeeFactorsInfrastructureFee:               NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.0005"),
		MarketFeeFactorsLiquidityFee:                    NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.001"),
		MarketAuctionMinimumDuration:                    NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("30m0s"),
		MarketAuctionMaximumDuration:                    NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate(week),
		MarketLiquidityBondPenaltyParameter:             NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1"),
		MarketLiquidityMaximumLiquidityFeeFactorLevel:   NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("1"),
		MarketLiquidityStakeToCCYSiskas:                 NewFloat(FloatGT(0)).Mutable(true).MustUpdate("1"),
		MarketLiquidityProvidersFeeDistribitionTimeStep: NewDuration(DurationGTE(0 * time.Second)).Mutable(true).MustUpdate("0s"),
		MarketLiquidityTargetStakeTriggeringRatio:       NewFloat(FloatGT(0)).Mutable(true).MustUpdate("0"),
		MarketTargetStakeTimeWindow:                     NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("1h0m0s"),
		MarketTargetStakeScalingFactor:                  NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("10"),
		MarketValueWindowLength:                         NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate(week),
		MarketPriceMonitoringDefaultParameters:          NewJSON(&proto.PriceMonitoringParameters{}, JSONProtoValidator()).Mutable(true).MustUpdate(`{"triggers": []}`),
		MarketPriceMonitoringUpdateFrequency:            NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("1m0s"),
		MarketLiquidityProvisionShapesMaxSize:           NewInt(IntGT(0)).Mutable(true).MustUpdate("100"),

		// governance market proposal
		GovernanceProposalMarketMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalMarketRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalMarketMinProposerBalance:    NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),
		GovernanceProposalMarketMinVoterBalance:       NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),

		// governance asset proposal
		GovernanceProposalAssetMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalAssetMinProposerBalance:    NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),
		GovernanceProposalAssetMinVoterBalance:       NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),

		// governance update market proposal
		GovernanceProposalUpdateMarketMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateMarketMinProposerBalance:    NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),
		GovernanceProposalUpdateMarketMinVoterBalance:       NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),

		// governance UpdateNetParam proposal
		GovernanceProposalUpdateNetParamMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateNetParamMinProposerBalance:    NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),
		GovernanceProposalUpdateNetParamMinVoterBalance:       NewInt(IntGTE(0)).Mutable(true).MustUpdate("0"),

		// no validation for this initially as we configure the
		// the bootstrapping asset.
		// validation will be added at node startup, so we can use dynamic stuff
		// e.g: assets and collateral when setting up a new ID.
		GovernanceVoteAsset: NewString().Mutable(true).MustUpdate("VOTE"),

		BlockchainsEthereumConfig: NewJSON(&proto.EthereumConfig{}, checks.EthereumConfig()).Mutable(true).
			MustUpdate("{\"network_id\": \"XXX\", \"chain_id\": \"XXX\", \"bridge_address\": \"0xXXX\", \"confirmations\": 3}"),
	}
}
