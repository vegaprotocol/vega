// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package netparams

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"code.vegaprotocol.io/vega/core/netparams/checks"
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
	gteU1  = UintGTE(num.NewUint(1))
	ltMaxU = UintLT(num.MaxUint())

	// Ints.
	gteI0    = IntGTE(0)
	gteI1    = IntGTE(1)
	lteI500  = IntLTE(500)
	lteI1000 = IntLTE(1000)

	// Durations.
	gte0s   = DurationGTE(0 * time.Second)
	gte1s   = DurationGTE(1 * time.Second)
	gte1m   = DurationGTE(1 * time.Minute)
	lte1d   = DurationLTE(24 * time.Hour)
	lte255h = DurationLTE(255 * time.Hour)
	lte1mo  = DurationLTE(30 * 24 * time.Hour)
	lte1y   = DurationLTE(365 * 24 * time.Hour)
)

func defaultNetParams() map[string]value {
	m := map[string]value{
		// markets
		MarketMarginScalingFactors:                      NewJSON(&proto.ScalingFactors{}, checks.MarginScalingFactor(), checks.MarginScalingFactorRange(num.DecimalOne(), num.DecimalFromInt64(100))).Mutable(true).MustUpdate(`{"search_level": 1.1, "initial_margin": 1.2, "collateral_release": 1.4}`),
		MarketFeeFactorsMakerFee:                        NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.00025"),
		MarketFeeFactorsInfrastructureFee:               NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.0005"),
		MarketAuctionMinimumDuration:                    NewDuration(gte1s, lte1d).Mutable(true).MustUpdate("30m0s"),
		MarketAuctionMaximumDuration:                    NewDuration(gte1s, lte1mo).Mutable(true).MustUpdate(week),
		MarketLiquidityBondPenaltyParameter:             NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("1"),
		MarketLiquidityMaximumLiquidityFeeFactorLevel:   NewDecimal(gtD0, lteD1).Mutable(true).MustUpdate("1"),
		MarketLiquidityStakeToCCYSiskas:                 NewDecimal(gteD0, lteD100).Mutable(true).MustUpdate("1"),
		MarketLiquidityProvidersFeeDistribitionTimeStep: NewDuration(gte0s, lte1mo).Mutable(true).MustUpdate("0s"),
		MarketLiquidityTargetStakeTriggeringRatio:       NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0"),
		MarketProbabilityOfTradingTauScaling:            NewDecimal(gteD1, lteD1000).Mutable(true).MustUpdate("1"),
		MarketMinProbabilityOfTradingForLPOrders:        NewDecimal(DecimalGTE(num.MustDecimalFromString("1e-12")), DecimalLTE(num.MustDecimalFromString("0.1"))).Mutable(true).MustUpdate("1e-8"),
		MarketTargetStakeTimeWindow:                     NewDuration(gte1s, lte1mo).Mutable(true).MustUpdate("1h0m0s"),
		MarketTargetStakeScalingFactor:                  NewDecimal(gtD0, lteD100).Mutable(true).MustUpdate("10"),
		MarketValueWindowLength:                         NewDuration(gte1m, lte1mo).Mutable(true).MustUpdate(week),
		MarketPriceMonitoringDefaultParameters:          NewJSON(&proto.PriceMonitoringParameters{}, checks.PriceMonitoringParametersAuctionExtension(5*time.Second, 30*24*time.Hour), checks.PriceMonitoringParametersHorizon(5*time.Second, 30*24*time.Hour), checks.PriceMonitoringParametersProbability(num.DecimalFromFloat(0.9), num.DecimalOne())).Mutable(true).MustUpdate(`{"triggers": []}`),
		MarketLiquidityProvisionShapesMaxSize:           NewInt(gteI1, lteI1000).Mutable(true).MustUpdate("5"),
		MarketMinLpStakeQuantumMultiple:                 NewDecimal(gtD0, DecimalLT(num.MustDecimalFromString("1e10"))).Mutable(true).MustUpdate("1"),
		RewardMarketCreationQuantumMultiple:             NewDecimal(gteD1, DecimalLT(num.MustDecimalFromString("1e20"))).Mutable(true).MustUpdate("10000000"),

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

		// spam protection policies
		SpamProtectionMaxVotes:               NewInt(gteI1).Mutable(true).MustUpdate("3"),
		SpamProtectionMinTokensForVoting:     NewDecimal(gteD1).Mutable(true).MustUpdate("100000000000000000000"),
		SpamProtectionMaxProposals:           NewInt(gteI1).Mutable(true).MustUpdate("3"),
		SpamProtectionMinTokensForProposal:   NewDecimal(gteD1).Mutable(true).MustUpdate("100000000000000000000000"),
		SpamProtectionMaxDelegations:         NewInt(gteI1).Mutable(true).MustUpdate("390"),
		SpamProtectionMinTokensForDelegation: NewDecimal(gteD1).Mutable(true).MustUpdate("1000000000000000000"),

		// no validation for this initially as we configure the
		// the bootstrapping asset.
		// validation will be added at node startup, so we can use dynamic stuff
		// e.g: assets and collateral when setting up a new ID.
		RewardAsset: NewString().Mutable(true).MustUpdate("VOTE"),

		BlockchainsEthereumConfig: NewJSON(&proto.EthereumConfig{}, types.CheckUntypedEthereumConfig).Mutable(true).
			MustUpdate("{\"network_id\": \"XXX\", \"chain_id\": \"XXX\", \"collateral_bridge_contract\": { \"address\": \"0xXXX\" }, \"confirmations\": 3, \"staking_bridge_contract\": { \"address\": \"0xXXX\", \"deployment_block_height\": 0}, \"token_vesting_contract\": { \"address\": \"0xXXX\", \"deployment_block_height\": 0 }, \"multisig_control_contract\": { \"address\": \"0xXXX\", \"deployment_block_height\": 0 }}"),

		ValidatorsEpochLength: NewDuration(gte1s, lte255h).Mutable(true).MustUpdate("24h0m0s"),

		ValidatorsVoteRequired: NewDecimal(gtD0, lteD1).Mutable(true).MustUpdate("0.67"),

		// network checkpoint parameters
		NetworkCheckpointTimeElapsedBetweenCheckpoints: NewDuration(DurationGT(0 * time.Second)).Mutable(true).MustUpdate("1m"),
		// take a snapshot every 1000 blocks, ~20 minutes
		// if we assume a block time of anything between 1 to 2 seconds
		SnapshotIntervalLength: NewInt(gteI0).Mutable(true).MustUpdate("1000"),

		FloatingPointUpdatesDuration: NewDuration().Mutable(true).MustUpdate("5m"),

		// validators by stake
		NumberOfTendermintValidators:               NewUint(gteU1, UintLTE(num.NewUint(500))).Mutable(true).MustUpdate("30"),
		ValidatorIncumbentBonus:                    NewDecimal(gteD0).Mutable(true).MustUpdate("1"),
		NumberEthMultisigSigners:                   NewUint(gteU1, UintLTE(num.NewUint(500))).Mutable(true).MustUpdate("13"),
		ErsatzvalidatorsRewardFactor:               NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.5"),
		MultipleOfTendermintValidatorsForEtsatzSet: NewDecimal(gteD0).Mutable(true).MustUpdate("0.5"),
		MinimumEthereumEventsForNewValidator:       NewUint(gteU0).Mutable(true).MustUpdate("3"),

		// transfers
		TransferFeeFactor:                  NewDecimal(gteD0, lteD1).Mutable(true).MustUpdate("0.001"),
		TransferMinTransferQuantumMultiple: NewDecimal(gteD0).Mutable(true).MustUpdate("0.1"),
		TransferMaxCommandsPerEpoch:        NewInt(gteI0).Mutable(true).MustUpdate("20"),

		// pow
		SpamPoWNumberOfPastBlocks:   NewUint(gteU1, UintLTE(num.NewUint(500))).Mutable(true).MustUpdate("100"),
		SpamPoWDifficulty:           NewUint(gteU0, UintLTE(num.NewUint(256))).Mutable(true).MustUpdate("15"),
		SpamPoWHashFunction:         NewString(checks.SpamPoWHashFunction([]string{crypto.Sha3})).Mutable(true).MustUpdate(crypto.Sha3),
		SpamPoWNumberOfTxPerBlock:   NewUint(gteU1).Mutable(true).MustUpdate("2"),
		SpamPoWIncreasingDifficulty: NewUint(gteU0).Mutable(true).MustUpdate("0"),

		LimitsProposeMarketEnabledFrom: NewString(checkOptionalRFC3339Date).Mutable(true).MustUpdate(""), // none by default
		LimitsProposeAssetEnabledFrom:  NewString(checkOptionalRFC3339Date).Mutable(true).MustUpdate(""), // none by default

		SpamProtectionMaxBatchSize: NewUint(UintGTE(num.NewUint(2)), UintLTE(num.NewUint(200))).Mutable(true).MustUpdate("15"),
		MaxGasPerBlock:             NewUint(UintGTE(num.NewUint(100)), UintLTE(num.NewUint(10000000))).Mutable(true).MustUpdate("100000"),
		DefaultGas:                 NewUint(UintGTE(num.NewUint(1)), UintLTE(num.NewUint(99))).Mutable(true).MustUpdate("1"),
		MaxPeggedOrders:            NewUint(UintGTE(num.NewUint(0)), UintLTE(num.NewUint(10000))).Mutable(true).MustUpdate("1500"),

		MarkPriceUpdateMaximumFrequency: NewDuration().Mutable(true).MustUpdate("5s"),
	}

	// add additional cross net param rules
	m[MarketAuctionMinimumDuration].AddRules(DurationDependentLT(MarketAuctionMaximumDuration, m[MarketAuctionMaximumDuration].(*Duration)))
	m[MarketAuctionMaximumDuration].AddRules(DurationDependentGT(MarketAuctionMinimumDuration, m[MarketAuctionMinimumDuration].(*Duration)))
	m[MarkPriceUpdateMaximumFrequency].AddRules(DurationGTE(time.Duration(0)))
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

func PriceMonitoringParametersValidation(i interface{}) error {
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
