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

package netparams

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/netparams/checks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

const (
	week = "168h0m0s"
)

var (
	// Decimals.
	gtD0     = DecimalGT(num.DecimalZero())
	gteD0    = DecimalGTE(num.DecimalZero())
	gteD1    = DecimalGTE(num.DecimalOne())
	lteD1    = DecimalLTE(num.DecimalOne())
	lteD100  = DecimalLTE(num.DecimalFromInt64(100))
	lteD1000 = DecimalLTE(num.DecimalFromInt64(1000))

	// Uints.
	gteU0  = UintGTE(num.UintZero())
	lteU1  = UintLTE(num.NewUint(1))
	gteU1  = UintGTE(num.NewUint(1))
	gteU5  = UintGTE(num.NewUint(5))
	ltMaxU = UintLT(num.MaxUint())

	// Ints.
	gteI0    = IntGTE(0)
	gteI1    = IntGTE(1)
	lteI500  = IntLTE(500)
	lteI1000 = IntLTE(1000)
	lteI1    = IntLTE(1)

	// Durations.
	gte0s   = DurationGTE(0 * time.Second)
	gte1s   = DurationGTE(1 * time.Second)
	gte1m   = DurationGTE(1 * time.Minute)
	lte1h   = DurationLTE(1 * time.Hour)
	lte1d   = DurationLTE(24 * time.Hour)
	lte255h = DurationLTE(255 * time.Hour)
	lte1mo  = DurationLTE(30 * 24 * time.Hour)
	lte1y   = DurationLTE(365 * 24 * time.Hour)
)

func defaultNetParams() map[string]value {
	m := map[string]value{
		// spots
		SpotMarketTradingEnabled: NewInt(gteI0, lteI1).Mutable(true).MustUpdate("0"),

		// perps
		PerpsMarketTradingEnabled: NewInt(gteI0, lteI1).Mutable(true).MustUpdate("0"),

		// ethereum oracles
		EthereumOraclesEnabled: NewInt(gteI0, lteI1).Mutable(true).MustUpdate("0"),

		// markets
		MarketMarginScalingFactors:                NewJSON(&proto.ScalingFactors{}, checks.MarginScalingFactor(), checks.MarginScalingFactorRange(num.DecimalOne(), num.DecimalFromInt64(100))).Mutable(true).MustUpdate(`{"search_level": 1.1, "initial_margin": 1.2, "collateral_release": 1.4}`),
		MarketFeeFactorsMakerFee:                  NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00025"),
		MarketFeeFactorsInfrastructureFee:         NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.0005"),
		MarketAuctionMinimumDuration:              NewDuration(gte1s, lte1d).Mutable(true).MustUpdate("30m0s"),
		MarketAuctionMaximumDuration:              NewDuration(gte1s, lte1mo).Mutable(true).MustUpdate(week),
		MarketLiquidityTargetStakeTriggeringRatio: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0"),
		MarketProbabilityOfTradingTauScaling:      NewDecimal(DecimalGTE(num.MustDecimalFromString("0.0001")), lteD1000).Mutable(true).MustUpdate("1"),
		MarketMinProbabilityOfTradingForLPOrders:  NewDecimal(DecimalGTE(num.MustDecimalFromString("1e-12")), DecimalLTE(num.MustDecimalFromString("0.1"))).Mutable(true).MustUpdate("1e-8"),
		MarketTargetStakeTimeWindow:               NewDuration(gte1s, lte1mo).Mutable(true).MustUpdate("1h0m0s"),
		MarketTargetStakeScalingFactor:            NewDecimal(gtD0, lteD100).Mutable(true).MustUpdate("10"),
		MarketValueWindowLength:                   NewDuration(gte1m, lte1mo).Mutable(true).MustUpdate(week),
		MarketPriceMonitoringDefaultParameters:    NewJSON(&proto.PriceMonitoringParameters{}, checks.PriceMonitoringParametersAuctionExtension(5*time.Second, 30*24*time.Hour), checks.PriceMonitoringParametersHorizon(5*time.Second, 30*24*time.Hour), checks.PriceMonitoringParametersProbability(num.DecimalFromFloat(0.9), num.DecimalOne())).Mutable(true).MustUpdate(`{"triggers": []}`),
		MarketLiquidityProvisionShapesMaxSize:     NewInt(gteI1, lteI1000).Mutable(true).MustUpdate("5"),
		MarketMinLpStakeQuantumMultiple:           NewDecimal(gtD0, DecimalLT(num.MustDecimalFromString("1e10"))).Mutable(true).MustUpdate("1"),
		RewardMarketCreationQuantumMultiple:       NewDecimal(gteD1, DecimalLT(num.MustDecimalFromString("1e20"))).Mutable(true).MustUpdate("10000000"),

		MarketLiquidityBondPenaltyParameter:              NewDecimal(gteD0, lteD1000).Mutable(true).MustUpdate("0.1"),
		MarketLiquidityEarlyExitPenalty:                  NewDecimal(gteD0, lteD1000).Mutable(true).MustUpdate("0.05"),
		MarketLiquidityMaximumLiquidityFeeFactorLevel:    NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("1"),
		MarketLiquiditySLANonPerformanceBondPenaltyMax:   NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.05"),
		MarketLiquiditySLANonPerformanceBondPenaltySlope: NewDecimal(gteD0, lteD1000).Mutable(true).MustUpdate("1"),
		MarketLiquidityStakeToCCYVolume:                  NewDecimal(gteD0, lteD100).Mutable(true).MustUpdate("1"),
		MarketLiquidityProvidersFeeCalculationTimeStep:   NewDuration(gte1s, lte255h).Mutable(true).MustUpdate("1m"),

		// governance market proposal
		GovernanceProposalMarketMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketMinEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalMarketMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalMarketRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalMarketRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalMarketMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalMarketMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),

		// governance asset proposal
		GovernanceProposalAssetMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetMinEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalAssetMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalAssetRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalAssetRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalAssetMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalAssetMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),

		// governance update asset proposal
		GovernanceProposalUpdateAssetMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateAssetMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateAssetMinEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateAssetMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateAssetRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateAssetRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateAssetMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalUpdateAssetMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),

		// governance update market proposal
		GovernanceProposalUpdateMarketMinClose:                   NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxClose:                   NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketMinEnact:                   NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateMarketMaxEnact:                   NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateMarketRequiredParticipation:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketRequiredMajority:           NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateMarketMinProposerBalance:         NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalUpdateMarketMinVoterBalance:            NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalUpdateMarketRequiredParticipationLP:    NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateMarketRequiredMajorityLP:         NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateMarketMinProposerEquityLikeShare: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),

		// governance UpdateNetParam proposal
		GovernanceProposalUpdateNetParamMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamMinEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalUpdateNetParamMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalUpdateNetParamRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalUpdateNetParamRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalUpdateNetParamMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalUpdateNetParamMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),

		// governance Freeform proposal
		GovernanceProposalFreeformMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalFreeformMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalFreeformRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalFreeformRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalFreeformMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalFreeformMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),

		// governance transfer proposal
		GovernanceProposalTransferMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalTransferMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalTransferRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00001"),
		GovernanceProposalTransferMinEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("48h0m0s"),
		GovernanceProposalTransferMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalTransferRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalTransferMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceProposalTransferMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1"),
		GovernanceTransferMaxAmount:                     NewDecimal(gteD1).Mutable(true).MustUpdate("7000"),
		GovernanceTransferMaxFraction:                   NewDecimal(gtD0, lteD1).Mutable(true).MustUpdate("1"),

		// Update referral program.
		GovernanceProposalReferralProgramMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("120h0m0s"),
		GovernanceProposalReferralProgramMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalReferralProgramMinEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("120h0m0s"),
		GovernanceProposalReferralProgramMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalReferralProgramRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.01"),
		GovernanceProposalReferralProgramRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalReferralProgramMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("10000000000000000000000"),
		GovernanceProposalReferralProgramMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1000000000000000000"),

		GovernanceProposalVolumeDiscountProgramMinClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("120h0m0s"),
		GovernanceProposalVolumeDiscountProgramMaxClose:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalVolumeDiscountProgramMinEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("120h0m0s"),
		GovernanceProposalVolumeDiscountProgramMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true).MustUpdate("8760h0m0s"),
		GovernanceProposalVolumeDiscountProgramRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.01"),
		GovernanceProposalVolumeDiscountProgramRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.66"),
		GovernanceProposalVolumeDiscountProgramMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("10000000000000000000000"),
		GovernanceProposalVolumeDiscountProgramMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true).MustUpdate("1000000000000000000"),

		// Delegation default params
		DelegationMinAmount: NewDecimal(gtD0).Mutable(true).MustUpdate("1"),

		// staking and delegation
		StakingAndDelegationRewardPayoutFraction: NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("1.0"),
		StakingAndDelegationRewardPayoutDelay:    NewDuration(DurationGTE(0 * time.Second)).Mutable(true).MustUpdate("24h0m0s"),

		StakingAndDelegationRewardMaxPayoutPerParticipant: NewDecimal(gteD0).Mutable(true).MustUpdate("0"),
		StakingAndDelegationRewardDelegatorShare:          NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.883"),
		StakingAndDelegationRewardMinimumValidatorStake:   NewDecimal(gteD0).Mutable(true).MustUpdate("0"),
		StakingAndDelegationRewardCompetitionLevel:        NewDecimal(gteD1).Mutable(true).MustUpdate("1.1"),
		StakingAndDelegationRewardMaxPayoutPerEpoch:       NewDecimal(gteD0).Mutable(true).MustUpdate("7000000000000000000000"),
		StakingAndDelegationRewardsMinValidators:          NewInt(gteI1, lteI500).Mutable(true).MustUpdate("5"),
		StakingAndDelegationRewardOptimalStakeMultiplier:  NewDecimal(gteD1).Mutable(true).MustUpdate("3.0"),

		// team rewards - //TODO review the constraint and defaults
		MinEpochsInTeamForMetricRewardEligibility: NewInt(gteI1, lteI500).Mutable(true).MustUpdate("5"),

		// spam protection policies
		SpamProtectionMaxVotes:                         NewInt(gteI1).Mutable(true).MustUpdate("3"),
		SpamProtectionMinTokensForVoting:               NewDecimal(gteD1).Mutable(true).MustUpdate("100000000000000000000"),
		SpamProtectionMaxProposals:                     NewInt(gteI1).Mutable(true).MustUpdate("3"),
		SpamProtectionMinTokensForProposal:             NewDecimal(gteD1).Mutable(true).MustUpdate("100000000000000000000000"),
		SpamProtectionMaxDelegations:                   NewInt(gteI1).Mutable(true).MustUpdate("390"),
		SpamProtectionMinTokensForDelegation:           NewDecimal(gteD1).Mutable(true).MustUpdate("1000000000000000000"),
		SpamProtectionMinimumWithdrawalQuantumMultiple: NewDecimal(gtD0, DecimalLT(num.MustDecimalFromString("1e6"))).Mutable(true).MustUpdate("10"),
		SpamProtectionMinMultisigUpdates:               NewDecimal(gteD1).Mutable(true).MustUpdate("100000000000000000000"),
		SpamProtectionMaxCreateReferralSet:             NewInt(gteI0).Mutable(true).MustUpdate("3"),
		SpamProtectionMaxUpdateReferralSet:             NewInt(gteI0).Mutable(true).MustUpdate("3"),
		SpamProtectionMaxApplyReferralCode:             NewInt(gteI0).Mutable(true).MustUpdate("5"),
		SpamProtectionBalanceSnapshotFrequency:         NewDuration(gte0s, lte1h).Mutable(true).MustUpdate("5s"),
		SpamProtectionApplyReferralMinFunds:            NewUint(UintGTE(num.NewUint(0))).Mutable(true).MustUpdate("10"),
		SpamProtectionReferralSetMinFunds:              NewUint(UintGTE(num.NewUint(0))).Mutable(true).MustUpdate("10"),
		SpamProtectionMaxUpdatePartyProfile:            NewInt(gteI0).Mutable(true).MustUpdate("10"),
		SpamProtectionUpdateProfileMinFunds:            NewUint(UintGTE(num.NewUint(0))).Mutable(true).MustUpdate("10"),

		// no validation for this initially as we configure the
		// the bootstrapping asset.
		// validation will be added at node startup, so we can use dynamic stuff
		// e.g: assets and collateral when setting up a new ID.
		RewardAsset: NewString().Mutable(true).MustUpdate("VOTE"),

		BlockchainsEthereumConfig: NewJSON(&proto.EthereumConfig{}, types.CheckUntypedEthereumConfig).Mutable(true).
			MustUpdate("{\"network_id\": \"XXX\", \"chain_id\": \"XXX\", \"collateral_bridge_contract\": { \"address\": \"0xXXX\" }, \"confirmations\": 3, \"staking_bridge_contract\": { \"address\": \"0xXXX\", \"deployment_block_height\": 0}, \"token_vesting_contract\": { \"address\": \"0xXXX\", \"deployment_block_height\": 0 }, \"multisig_control_contract\": { \"address\": \"0xXXX\", \"deployment_block_height\": 0 }}"),
		BlockchainsEthereumL2Configs: NewJSON(&proto.EthereumL2Configs{}, types.CheckUntypedEthereumL2Configs).Mutable(true).
			MustUpdate(
				`{"configs":[{"network_id":"100","chain_id":"100","confirmations":3,"name":"Gnosis Chain", "block_interval": 3}, {"network_id":"42161","chain_id":"42161","confirmations":3,"name":"Arbitrum One", "block_interval": 50}]}`,
			),

		ValidatorsEpochLength: NewDuration(gte1s, lte255h).Mutable(true).MustUpdate("24h0m0s"),

		ValidatorsVoteRequired: NewDecimal(gtD0, lteD1).Mutable(true).MustUpdate("0.67"),

		// network checkpoint parameters
		NetworkCheckpointTimeElapsedBetweenCheckpoints: NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("1m"),
		// take a snapshot every 1000 blocks, ~20 minutes
		// if we assume a block time of anything between 1 to 2 seconds
		SnapshotIntervalLength: NewUint(gteU1).Mutable(true).MustUpdate("1000"),

		FloatingPointUpdatesDuration: NewDuration(DurationGTE(10*time.Second), DurationLTE(1*time.Hour)).Mutable(true).MustUpdate("5m"),

		// validators by stake
		NumberOfTendermintValidators:               NewUint(gteU1, UintLTE(num.NewUint(500))).Mutable(true).MustUpdate("30"),
		ValidatorIncumbentBonus:                    NewDecimal(gteD0).Mutable(true).MustUpdate("1"),
		NumberEthMultisigSigners:                   NewUint(gteU1, UintLTE(num.NewUint(500))).Mutable(true).MustUpdate("13"),
		ErsatzvalidatorsRewardFactor:               NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.5"),
		MultipleOfTendermintValidatorsForEtsatzSet: NewDecimal(gteD0).Mutable(true).MustUpdate("0.5"),
		MinimumEthereumEventsForNewValidator:       NewUint(gteU0).Mutable(true).MustUpdate("3"),

		// transfers
		TransferFeeFactor:                       NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.001"),
		TransferMinTransferQuantumMultiple:      NewDecimal(gteD0).Mutable(true).MustUpdate("0.1"),
		TransferMaxCommandsPerEpoch:             NewInt(gteI0).Mutable(true).MustUpdate("20"),
		TransferFeeMaxQuantumAmount:             NewDecimal(gteD0).Mutable(true).MustUpdate("1"),
		TransferFeeDiscountDecayFraction:        NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.8"),
		TransferFeeDiscountMinimumTrackedAmount: NewDecimal(gteD0).Mutable(true).MustUpdate("0.01"),

		// pow
		SpamPoWNumberOfPastBlocks:   NewUint(gteU5, UintLTE(num.NewUint(500))).Mutable(true).MustUpdate("100"),
		SpamPoWDifficulty:           NewUint(gteU0, UintLTE(num.NewUint(256))).Mutable(true).MustUpdate("15"),
		SpamPoWHashFunction:         NewString(checks.SpamPoWHashFunction([]string{crypto.Sha3})).Mutable(true).MustUpdate(crypto.Sha3),
		SpamPoWNumberOfTxPerBlock:   NewUint(gteU1).Mutable(true).MustUpdate("2"),
		SpamPoWIncreasingDifficulty: NewUint(gteU0, lteU1).Mutable(true).MustUpdate("0"),

		LimitsProposeMarketEnabledFrom: NewString(checkOptionalRFC3339Date).Mutable(true).MustUpdate(""), // none by default
		LimitsProposeAssetEnabledFrom:  NewString(checkOptionalRFC3339Date).Mutable(true).MustUpdate(""), // none by default

		SpamProtectionMaxBatchSize: NewUint(UintGTE(num.NewUint(2)), UintLTE(num.NewUint(200))).Mutable(true).MustUpdate("15"),
		MaxGasPerBlock:             NewUint(UintGTE(num.NewUint(100)), UintLTE(num.NewUint(10000000))).Mutable(true).MustUpdate("500000"),
		DefaultGas:                 NewUint(UintGTE(num.NewUint(1)), UintLTE(num.NewUint(99))).Mutable(true).MustUpdate("1"),
		MinBlockCapacity:           NewUint(UintGTE(num.NewUint(1)), UintLTE(num.NewUint(10000))).Mutable(true).MustUpdate("32"),
		MaxPeggedOrders:            NewUint(UintGTE(num.NewUint(0)), UintLTE(num.NewUint(10000))).Mutable(true).MustUpdate("1500"),

		MarkPriceUpdateMaximumFrequency:       NewDuration(gte0s, lte1h).Mutable(true).MustUpdate("5s"),
		InternalCompositePriceUpdateFrequency: NewDuration(gte0s, lte1h).Mutable(true).MustUpdate("5s"),
		ValidatorPerformanceScalingFactor:     NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0"),
		MarketSuccessorLaunchWindow:           NewDuration(gte1s, lte1mo).Mutable(true).MustUpdate("168h"), // 168h == 7 days
		SpamProtectionMaxStopOrdersPerMarket:  NewUint(UintGTE(num.UintZero()), UintLTE(num.NewUint(100))).Mutable(true).MustUpdate("4"),

		RewardsVestingBaseRate:        NewDecimal(gtD0, lteD1).Mutable(true).MustUpdate("0.25"),
		RewardsVestingMinimumTransfer: NewDecimal(gtD0).Mutable(true).MustUpdate("10"),
		RewardsVestingBenefitTiers:    NewJSON(&proto.VestingBenefitTiers{}, types.CheckUntypedVestingBenefitTier).Mutable(true).MustUpdate(`{"tiers": []}`),

		// Referral program
		ReferralProgramMaxReferralTiers:                        NewUint(UintGTE(num.NewUint(0)), UintLTE(num.NewUint(100))).Mutable(true).MustUpdate("10"),
		ReferralProgramMaxReferralRewardFactor:                 NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.5"),
		ReferralProgramMaxReferralDiscountFactor:               NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.5"),
		ReferralProgramMaxPartyNotionalVolumeByQuantumPerEpoch: NewUint(UintGTE(num.NewUint(0))).Mutable(true).MustUpdate("250000"),
		ReferralProgramMinStakedVegaTokens:                     NewUint(UintGTE(num.NewUint(0))).Mutable(true).MustUpdate("0"),
		ReferralProgramMaxReferralRewardProportion:             NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.5"),

		VolumeDiscountProgramMaxVolumeDiscountFactor: NewDecimal(gteD0, DecimalLTE(num.MustDecimalFromString("1"))).Mutable(true).MustUpdate("0.9"),
		VolumeDiscountProgramMaxBenefitTiers:         NewUint(UintGTE(num.NewUint(0)), UintLTE(num.NewUint(10))).Mutable(true).MustUpdate("10"),

		RewardsActivityStreakInactivityLimit:       NewUint(UintGTE(num.UintOne()), UintLTE(num.NewUint(100))).Mutable(true).MustUpdate("3"),
		RewardsActivityStreakBenefitTiers:          NewJSON(&proto.ActivityStreakBenefitTiers{}, types.CheckUntypedActivityStreakBenefitTier).Mutable(true).MustUpdate(`{"tiers": []}`),
		RewardsActivityStreakMinQuantumOpenVolume:  NewUint().Mutable(true).MustUpdate("500"),
		RewardsActivityStreakMinQuantumTradeVolume: NewUint().Mutable(true).MustUpdate("2500"),
	}

	// add additional cross net param rules
	m[MarketAuctionMinimumDuration].AddRules(DurationDependentLT(MarketAuctionMaximumDuration, m[MarketAuctionMaximumDuration].(*Duration)))
	m[MarketAuctionMaximumDuration].AddRules(DurationDependentGT(MarketAuctionMinimumDuration, m[MarketAuctionMinimumDuration].(*Duration)))

	m[NumberEthMultisigSigners].AddRules(UintDependentLTE(NumberOfTendermintValidators, m[NumberOfTendermintValidators].(*Uint), num.MustDecimalFromString("1")))
	m[NumberOfTendermintValidators].AddRules(UintDependentGTE(NumberEthMultisigSigners, m[NumberEthMultisigSigners].(*Uint), num.MustDecimalFromString("1")))

	// ensure that MinBlockCapacity <= 2*
	m[MaxGasPerBlock].AddRules(UintDependentGTE(MinBlockCapacity, m[MinBlockCapacity].(*Uint), num.MustDecimalFromString("2")))
	m[MinBlockCapacity].AddRules(UintDependentLTE(MaxGasPerBlock, m[MaxGasPerBlock].(*Uint), num.MustDecimalFromString("0.5")))
	m[MarkPriceUpdateMaximumFrequency].AddRules(DurationGT(time.Duration(0)))
	// could just do 24 * 3600 * time.Second, but this is easier to read
	maxFreq, _ := time.ParseDuration("24h")
	m[MarkPriceUpdateMaximumFrequency].AddRules(DurationGT(time.Duration(0)), DurationLTE(maxFreq))

	m[InternalCompositePriceUpdateFrequency].AddRules(DurationGT(time.Duration(0)))
	// could just do 24 * 3600 * time.Second, but this is easier to read
	m[InternalCompositePriceUpdateFrequency].AddRules(DurationGT(time.Duration(0)), DurationLTE(maxFreq))

	m[MarketLiquidityProvidersFeeCalculationTimeStep].AddRules(
		DurationDependentLTE(ValidatorsEpochLength, m[ValidatorsEpochLength].(*Duration)),
	)
	return m
}

func checkOptionalRFC3339Date(d string) error {
	if len(d) <= 0 {
		// an empty string is correct, it just disable the value.
		return nil
	}

	// now let's just try to parse and see
	_, err := time.Parse(time.RFC3339, d)
	return err
}

func PriceMonitoringParametersValidation(i interface{}, _ interface{}) error {
	pmp, ok := i.(*proto.PriceMonitoringParameters)
	if !ok {
		return errors.New("not a price monitoring parameters type")
	}

	for _, trigger := range pmp.Triggers {
		if trigger.Horizon <= 0 {
			return fmt.Errorf("triggers.horizon must be greater than `0`, got `%d`", trigger.Horizon)
		}

		probability, err := num.DecimalFromString(trigger.Probability)
		if err != nil {
			return fmt.Errorf("triggers.probability must be greater than `0`, got `%s`", trigger.Probability)
		}

		if probability.Cmp(num.DecimalZero()) <= 0 {
			return fmt.Errorf("triggers.probability must be greater than `0`, got `%s`", trigger.Probability)
		}
		if probability.Cmp(num.DecimalFromInt64(1)) >= 0 {
			return fmt.Errorf("triggers.probability must be lower than `1`, got `%s`", trigger.Probability)
		}

		if trigger.AuctionExtension <= 0 {
			return fmt.Errorf("triggers.auction_extension must be greater than `0`, got `%d`", trigger.AuctionExtension)
		}
	}

	return nil
}
