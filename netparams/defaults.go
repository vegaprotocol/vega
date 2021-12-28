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
		MarketFeeFactorsMakerFee:                        NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.00025"),
		MarketFeeFactorsInfrastructureFee:               NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.0005"),
		MarketAuctionMinimumDuration:                    NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("30m0s"),
		MarketAuctionMaximumDuration:                    NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate(week),
		MarketLiquidityBondPenaltyParameter:             NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("1"),
		MarketLiquidityMaximumLiquidityFeeFactorLevel:   NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("1"),
		MarketLiquidityStakeToCCYSiskas:                 NewDecimal(DecimalGT(num.DecimalZero())).Mutable(true).MustUpdate("1"),
		MarketLiquidityProvidersFeeDistribitionTimeStep: NewDuration(DurationGTE(0 * time.Second)).Mutable(true).MustUpdate("0s"),
		MarketLiquidityTargetStakeTriggeringRatio:       NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("0"),
		MarketProbabilityOfTradingTauScaling:            NewDecimal(DecimalGTE(num.MustDecimalFromString("1."))).Mutable(true).MustUpdate("1"),
		MarketMinProbabilityOfTradingForLPOrders:        NewDecimal(DecimalGTE(num.MustDecimalFromString("1e-15")), DecimalLTE(num.MustDecimalFromString("0.1"))).Mutable(true).MustUpdate("1e-8"),
		MarketTargetStakeTimeWindow:                     NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("1h0m0s"),
		MarketTargetStakeScalingFactor:                  NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("10"),
		MarketValueWindowLength:                         NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate(week),
		MarketPriceMonitoringDefaultParameters:          NewJSON(&proto.PriceMonitoringParameters{}, JSONProtoValidator()).Mutable(true).MustUpdate(`{"triggers": []}`),
		MarketPriceMonitoringUpdateFrequency:            NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("1m0s"),
		MarketLiquidityProvisionShapesMaxSize:           NewInt(IntGT(0)).Mutable(true).MustUpdate("100"),

		// governance market proposal
		GovernanceProposalMarketMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketRequiredParticipation: NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalMarketRequiredMajority:      NewDecimal(DecimalGTE(num.MustDecimalFromString("0.5")), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalMarketMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalMarketMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance asset proposal
		GovernanceProposalAssetMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetRequiredParticipation: NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetRequiredMajority:      NewDecimal(DecimalGTE(num.MustDecimalFromString("0.5")), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalAssetMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalAssetMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance update market proposal
		GovernanceProposalUpdateMarketMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketRequiredParticipation: NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketRequiredMajority:      NewDecimal(DecimalGTE(num.MustDecimalFromString("0.5")), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateMarketMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalUpdateMarketMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance UpdateNetParam proposal
		GovernanceProposalUpdateNetParamMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamMinEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxEnact:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamRequiredParticipation: NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamRequiredMajority:      NewDecimal(DecimalGTE(num.MustDecimalFromString("0.5")), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateNetParamMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalUpdateNetParamMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// governance Freeform proposal
		GovernanceProposalFreeformMinClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalFreeformMaxClose:              NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalFreeformRequiredParticipation: NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalFreeformRequiredMajority:      NewDecimal(DecimalGTE(num.MustDecimalFromString("0.5")), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalFreeformMinProposerBalance:    NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),
		GovernanceProposalFreeformMinVoterBalance:       NewUint(UintGTE(num.Zero())).Mutable(true).MustUpdate("0"),

		// Delegation default params
		DelegationMinAmount: NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("1"),

		// staking and delegation
		StakingAndDelegationRewardPayoutFraction:          NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("1.0"),
		StakingAndDelegationRewardMaxPayoutPerParticipant: NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("0"),
		StakingAndDelegationRewardPayoutDelay:             NewDuration(DurationGTE(0 * time.Second)).Mutable(true).MustUpdate("24h0m0s"),
		StakingAndDelegationRewardDelegatorShare:          NewDecimal(DecimalGTE(num.DecimalZero()), DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.883"),
		StakingAndDelegationRewardMinimumValidatorStake:   NewDecimal(DecimalGTE(num.DecimalZero())).Mutable(true).MustUpdate("0"),
		StakingAndDelegationRewardCompetitionLevel:        NewDecimal(DecimalGT(num.MustDecimalFromString("1")), DecimalLTE(num.MustDecimalFromString("1000"))).Mutable(true).MustUpdate("1.1"),
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

		ValidatorsVoteRequired: NewDecimal(DecimalGTE(num.MustDecimalFromString("0.67")), DecimalLTE(num.MustDecimalFromString("1.0"))).Mutable(true).MustUpdate("0.67"),

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
