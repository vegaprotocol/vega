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
	lteU1  = UintLTE(num.NewUint(1))
	gteU1  = UintGTE(num.NewUint(1))
	gteU5  = UintGTE(num.NewUint(5))
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
	lte1h   = DurationLTE(1 * time.Hour)
	lte1d   = DurationLTE(24 * time.Hour)
	lte255h = DurationLTE(255 * time.Hour)
	lte1mo  = DurationLTE(30 * 24 * time.Hour)
	lte1y   = DurationLTE(365 * 24 * time.Hour)
)

func defaultNetParams() map[string]value {
	m := map[string]value{
		// markets
		MarketMarginScalingFactors:                      NewJSON(&proto.ScalingFactors{}, checks.MarginScalingFactor(), checks.MarginScalingFactorRange(num.DecimalOne(), num.DecimalFromInt64(100))).Mutable(true),
		MarketFeeFactorsMakerFee:                        NewDecimal(gteD0, lteD1).Mutable(true),
		MarketFeeFactorsInfrastructureFee:               NewDecimal(gteD0, lteD1).Mutable(true),
		MarketAuctionMinimumDuration:                    NewDuration(gte1s, lte1d).Mutable(true),
		MarketAuctionMaximumDuration:                    NewDuration(gte1s, lte1mo).Mutable(true),
		MarketLiquidityBondPenaltyParameter:             NewDecimal(gteD0, lteD1).Mutable(true),
		MarketLiquidityMaximumLiquidityFeeFactorLevel:   NewDecimal(gtD0, lteD1).Mutable(true),
		MarketLiquidityStakeToCCYVolume:                 NewDecimal(gteD0, lteD100).Mutable(true),
		MarketLiquidityProvidersFeeDistribitionTimeStep: NewDuration(gte0s, lte1mo).Mutable(true),
		MarketLiquidityTargetStakeTriggeringRatio:       NewDecimal(gteD0, lteD1).Mutable(true),
		MarketProbabilityOfTradingTauScaling:            NewDecimal(gteD1, lteD1000).Mutable(true),
		MarketMinProbabilityOfTradingForLPOrders:        NewDecimal(DecimalGTE(num.MustDecimalFromString("1e-12")), DecimalLTE(num.MustDecimalFromString("0.1"))).Mutable(true),
		MarketTargetStakeTimeWindow:                     NewDuration(gte1s, lte1mo).Mutable(true),
		MarketTargetStakeScalingFactor:                  NewDecimal(gtD0, lteD100).Mutable(true),
		MarketValueWindowLength:                         NewDuration(gte1m, lte1mo).Mutable(true),
		MarketPriceMonitoringDefaultParameters:          NewJSON(&proto.PriceMonitoringParameters{}, checks.PriceMonitoringParametersAuctionExtension(5*time.Second, 30*24*time.Hour), checks.PriceMonitoringParametersHorizon(5*time.Second, 30*24*time.Hour), checks.PriceMonitoringParametersProbability(num.DecimalFromFloat(0.9), num.DecimalOne())).Mutable(true),
		MarketLiquidityProvisionShapesMaxSize:           NewInt(gteI1, lteI1000).Mutable(true),
		MarketMinLpStakeQuantumMultiple:                 NewDecimal(gtD0, DecimalLT(num.MustDecimalFromString("1e10"))).Mutable(true),
		RewardMarketCreationQuantumMultiple:             NewDecimal(gteD1, DecimalLT(num.MustDecimalFromString("1e20"))).Mutable(true),

		// governance market proposal
		GovernanceProposalMarketMinClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalMarketMaxClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalMarketMinEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalMarketMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalMarketRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalMarketRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalMarketMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true),
		GovernanceProposalMarketMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true),

		// governance asset proposal
		GovernanceProposalAssetMinClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalAssetMaxClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalAssetMinEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalAssetMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalAssetRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalAssetRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalAssetMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true),
		GovernanceProposalAssetMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true),

		// governance update asset proposal
		GovernanceProposalUpdateAssetMinClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateAssetMaxClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateAssetMinEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateAssetMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateAssetRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateAssetRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateAssetMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true),
		GovernanceProposalUpdateAssetMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true),

		// governance update market proposal
		GovernanceProposalUpdateMarketMinClose:                   NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateMarketMaxClose:                   NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateMarketMinEnact:                   NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateMarketMaxEnact:                   NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateMarketRequiredParticipation:      NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateMarketRequiredMajority:           NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateMarketMinProposerBalance:         NewUint(gteU1, ltMaxU).Mutable(true),
		GovernanceProposalUpdateMarketMinVoterBalance:            NewUint(gteU1, ltMaxU).Mutable(true),
		GovernanceProposalUpdateMarketRequiredParticipationLP:    NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateMarketRequiredMajorityLP:         NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateMarketMinProposerEquityLikeShare: NewDecimal(gteD0, lteD1).Mutable(true),

		// governance UpdateNetParam proposal
		GovernanceProposalUpdateNetParamMinClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateNetParamMaxClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateNetParamMinEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateNetParamMaxEnact:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalUpdateNetParamRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateNetParamRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalUpdateNetParamMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true),
		GovernanceProposalUpdateNetParamMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true),

		// governance Freeform proposal
		GovernanceProposalFreeformMinClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalFreeformMaxClose:              NewDuration(gte1s, lte1y).Mutable(true),
		GovernanceProposalFreeformRequiredParticipation: NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalFreeformRequiredMajority:      NewDecimal(gteD0, lteD1).Mutable(true),
		GovernanceProposalFreeformMinProposerBalance:    NewUint(gteU1, ltMaxU).Mutable(true),
		GovernanceProposalFreeformMinVoterBalance:       NewUint(gteU1, ltMaxU).Mutable(true),

		// Delegation default params
		DelegationMinAmount: NewDecimal(gtD0).Mutable(true),

		// staking and delegation
		StakingAndDelegationRewardPayoutFraction: NewDecimal(gteD0, lteD1).Mutable(true),
		StakingAndDelegationRewardPayoutDelay:    NewDuration(DurationGTE(0 * time.Second)).Mutable(true),

		StakingAndDelegationRewardMaxPayoutPerParticipant: NewDecimal(gteD0).Mutable(true),
		StakingAndDelegationRewardDelegatorShare:          NewDecimal(gteD0, lteD1).Mutable(true),
		StakingAndDelegationRewardMinimumValidatorStake:   NewDecimal(gteD0).Mutable(true),
		StakingAndDelegationRewardCompetitionLevel:        NewDecimal(gteD1).Mutable(true),
		StakingAndDelegationRewardMaxPayoutPerEpoch:       NewDecimal(gteD0).Mutable(true),
		StakingAndDelegationRewardsMinValidators:          NewInt(gteI1, lteI500).Mutable(true),
		StakingAndDelegationRewardOptimalStakeMultiplier:  NewDecimal(gteD1).Mutable(true),

		// spam protection policies
		SpamProtectionMaxVotes:                         NewInt(gteI1).Mutable(true),
		SpamProtectionMinTokensForVoting:               NewDecimal(gteD1).Mutable(true),
		SpamProtectionMaxProposals:                     NewInt(gteI1).Mutable(true),
		SpamProtectionMinTokensForProposal:             NewDecimal(gteD1).Mutable(true),
		SpamProtectionMaxDelegations:                   NewInt(gteI1).Mutable(true),
		SpamProtectionMinTokensForDelegation:           NewDecimal(gteD1).Mutable(true),
		SpamProtectionMinimumWithdrawalQuantumMultiple: NewDecimal(gtD0, DecimalLT(num.MustDecimalFromString("1e6"))).Mutable(true),
		SpamProtectionMinMultisigUpdates:               NewDecimal(gteD1).Mutable(true),
		// no validation for this initially as we configure the
		// the bootstrapping asset.
		// validation will be added at node startup, so we can use dynamic stuff
		// e.g: assets and collateral when setting up a new ID.
		RewardAsset: NewString().Mutable(true).MustUpdate("VOTE"),

		BlockchainsEthereumConfig: NewJSON(&proto.EthereumConfig{}, types.CheckUntypedEthereumConfig).Mutable(true),

		ValidatorsEpochLength: NewDuration(gte1s, lte255h).Mutable(true),

		ValidatorsVoteRequired: NewDecimal(gtD0, lteD1).Mutable(true),

		// network checkpoint parameters
		NetworkCheckpointTimeElapsedBetweenCheckpoints: NewDuration(DurationGT(0 * time.Second)).Mutable(true),
		// take a snapshot every 1000 blocks, ~20 minutes
		// if we assume a block time of anything between 1 to 2 seconds
		SnapshotIntervalLength: NewInt(gteI0).Mutable(true),

		FloatingPointUpdatesDuration: NewDuration(DurationGTE(10*time.Second), DurationLTE(1*time.Hour)).Mutable(true),

		// validators by stake
		NumberOfTendermintValidators:               NewUint(gteU1, UintLTE(num.NewUint(500))).Mutable(true),
		ValidatorIncumbentBonus:                    NewDecimal(gteD0).Mutable(true),
		NumberEthMultisigSigners:                   NewUint(gteU1, UintLTE(num.NewUint(500))).Mutable(true),
		ErsatzvalidatorsRewardFactor:               NewDecimal(gteD0, lteD1).Mutable(true),
		MultipleOfTendermintValidatorsForEtsatzSet: NewDecimal(gteD0).Mutable(true),
		MinimumEthereumEventsForNewValidator:       NewUint(gteU0).Mutable(true),

		// transfers
		TransferFeeFactor:                  NewDecimal(gteD0, lteD1).Mutable(true),
		TransferMinTransferQuantumMultiple: NewDecimal(gteD0).Mutable(true),
		TransferMaxCommandsPerEpoch:        NewInt(gteI0).Mutable(true),

		// pow
		SpamPoWNumberOfPastBlocks:   NewUint(gteU5, UintLTE(num.NewUint(500))).Mutable(true),
		SpamPoWDifficulty:           NewUint(gteU0, UintLTE(num.NewUint(256))).Mutable(true),
		SpamPoWHashFunction:         NewString(checks.SpamPoWHashFunction([]string{crypto.Sha3})).Mutable(true),
		SpamPoWNumberOfTxPerBlock:   NewUint(gteU1).Mutable(true),
		SpamPoWIncreasingDifficulty: NewUint(gteU0, lteU1).Mutable(true),

		LimitsProposeMarketEnabledFrom: NewString(checkOptionalRFC3339Date).Mutable(true),// none by default
		LimitsProposeAssetEnabledFrom:  NewString(checkOptionalRFC3339Date).Mutable(true), // none by default

		SpamProtectionMaxBatchSize: NewUint(UintGTE(num.NewUint(2)), UintLTE(num.NewUint(200))).Mutable(true),
		MaxGasPerBlock:             NewUint(UintGTE(num.NewUint(100)), UintLTE(num.NewUint(10000000))).Mutable(true),
		DefaultGas:                 NewUint(UintGTE(num.NewUint(1)), UintLTE(num.NewUint(99))).Mutable(true),
		MinBlockCapacity:           NewUint(UintGTE(num.NewUint(1)), UintLTE(num.NewUint(10000))).Mutable(true),
		MaxPeggedOrders:            NewUint(UintGTE(num.NewUint(0)), UintLTE(num.NewUint(10000))).Mutable(true),

		MarkPriceUpdateMaximumFrequency:   NewDuration(gte0s, lte1h).Mutable(true),
		ValidatorPerformanceScalingFactor: NewDecimal(gteD0, lteD1).Mutable(true),
	}

	// add additional cross net param rules
	m[MarketAuctionMinimumDuration].AddRules(DurationDependentLT(MarketAuctionMaximumDuration, m[MarketAuctionMaximumDuration].(*Duration)))
	m[MarketAuctionMaximumDuration].AddRules(DurationDependentGT(MarketAuctionMinimumDuration, m[MarketAuctionMinimumDuration].(*Duration)))

	m[NumberEthMultisigSigners].AddRules(UintDependentLTE(NumberOfTendermintValidators, m[NumberOfTendermintValidators].(*Uint), num.MustDecimalFromString("1")))
	m[NumberOfTendermintValidators].AddRules(UintDependentGTE(NumberEthMultisigSigners, m[NumberEthMultisigSigners].(*Uint), num.MustDecimalFromString("1")))

	// ensure that MinBlockCapacity <= 2*
	m[MaxGasPerBlock].AddRules(UintDependentGTE(MinBlockCapacity, m[MinBlockCapacity].(*Uint), num.MustDecimalFromString("2")))
	m[MinBlockCapacity].AddRules(UintDependentLTE(MaxGasPerBlock, m[MaxGasPerBlock].(*Uint), num.MustDecimalFromString("0.5")))
	m[MarkPriceUpdateMaximumFrequency].AddRules(DurationGTE(time.Duration(0)))
	// could just do 24 * 3600 * time.Second, but this is easier to read
	maxFreq, _ := time.ParseDuration("24h")
	m[MarkPriceUpdateMaximumFrequency].AddRules(DurationGTE(time.Duration(0)), DurationLTE(maxFreq))
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
