package netparams

import (
	"time"

	"code.vegaprotocol.io/vega/types/num"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/netparams/checks"
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
		MarketAuctionMinimumDuration:                    NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("30m0s"),
		MarketAuctionMaximumDuration:                    NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate(week),
		MarketLiquidityBondPenaltyParameter:             NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("1"),
		MarketLiquidityMaximumLiquidityFeeFactorLevel:   NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("1"),
		MarketLiquidityStakeToCCYSiskas:                 NewFloat(FloatGT(0)).Mutable(true).MustUpdate("1"),
		MarketLiquidityProvidersFeeDistribitionTimeStep: NewDuration(DurationGTE(0 * time.Second)).Mutable(true).MustUpdate("0s"),
		MarketLiquidityTargetStakeTriggeringRatio:       NewFloat(FloatGTE(0)).Mutable(true).MustUpdate("0"),
		MarketProbabilityOfTradingTauScaling:            NewFloat(FloatGTE(1.)).Mutable(true).MustUpdate("1"),
		MarketMinProbabilityOfTradingForLPOrders:        NewFloat(FloatGTE(1e-15), FloatLTE(0.1)).Mutable(true).MustUpdate("1e-8"),
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
		GovernanceProposalMarketMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalMarketMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance asset proposal
		GovernanceProposalAssetMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalAssetMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalAssetMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance update market proposal
		GovernanceProposalUpdateMarketMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateMarketMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalUpdateMarketMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance UpdateNetParam proposal
		GovernanceProposalUpdateNetParamMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateNetParamMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalUpdateNetParamMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance Freeform proposal
		GovernanceProposalFreeformMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalFreeformMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalFreeformRequiredParticipation: NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalFreeformRequiredMajority:      NewFloat(FloatGTE(0.5), FloatLTE(1)).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalFreeformMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalFreeformMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// Delegation default params
		DelegationMinAmount: NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("1"),

		// staking and delegation
		StakingAndDelegationRewardPayoutFraction:          NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("1.0"),
		StakingAndDelegationRewardMaxPayoutPerParticipant: NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("0"),
		StakingAndDelegationRewardPayoutDelay:             NewDuration(DurationGTE(0 * time.Second)).Mutable(true).MustUpdate("24h0m0s"),
		StakingAndDelegationRewardDelegatorShare:          NewFloat(FloatGTE(0), FloatLTE(1)).Mutable(true).MustUpdate("0.883"),
		StakingAndDelegationRewardMinimumValidatorStake:   NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("0"),
		StakingAndDelegationRewardCompetitionLevel:        NewFloat(FloatGT(1), FloatLTE(1000)).Mutable(true).MustUpdate("1.1"),
		StakingAndDelegationRewardMaxPayoutPerEpoch:       NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("7000000000000000000000"),
		StakingAndDelegationRewardsMinValidators:          NewInt(IntGTE(1)).Mutable(true).MustUpdate("5"),
		StakingAndDelegationRewardOptimalStakeMultiplier:  NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("3.0"),

		// spam protection policies
		SpamProtectionMaxVotes:               NewInt(IntGTE(0)).Mutable(true).MustUpdate("3"),
		SpamProtectionMinTokensForVoting:     NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("100000000000000000000"),
		SpamProtectionMaxProposals:           NewInt(IntGTE(0)).Mutable(true).MustUpdate("3"),
		SpamProtectionMinTokensForProposal:   NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("100000000000000000000000"),
		SpamProtectionMaxDelegations:         NewInt(IntGTE(0)).Mutable(true).MustUpdate("390"),
		SpamProtectionMinTokensForDelegation: NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("1000000000000000000"),

		// no validation for this initially as we configure the
		// the bootstrapping asset.
		// validation will be added at node startup, so we can use dynamic stuff
		// e.g: assets and collateral when setting up a new ID.
		RewardAsset: NewString().Mutable(true).MustUpdate("VOTE"),

		BlockchainsEthereumConfig: NewJSON(&proto.EthereumConfig{}, checks.EthereumConfig()).Mutable(true).
			MustUpdate("{\"network_id\": \"XXX\", \"chain_id\": \"XXX\", \"bridge_address\": \"0xXXX\", \"confirmations\": 3, \"staking_bridge_addresses\": []}"),

		ValidatorsEpochLength: NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("24h0m0s"),

		ValidatorsVoteRequired: NewFloat(FloatGTE(0.67), FloatLTE(1.0)).Mutable(true).MustUpdate("0.67"),

		// @TODO add watcher for NetworkEOL > MarketFreezeDate
		// network checkpoint parameters
		NetworkCheckpointMarketFreezeDate:              NewTime().Mutable(true).MustUpdate("never"),
		NetworkCheckpointNetworkEOLDate:                NewTime().Mutable(true).MustUpdate("never"),
		NetworkCheckpointTimeElapsedBetweenCheckpoints: NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("1m"),
		// take a snapshot every N blocks, default to every block for now (debug)
		SnapshotIntervalLength: NewInt(IntGTE(0)).Mutable(true).MustUpdate("1"),

		FloatingPointUpdatesDuration: NewDuration().Mutable(true).MustUpdate("5m"),
	}
}
