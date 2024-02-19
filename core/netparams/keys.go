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

const (
	SpotMarketTradingEnabled  = "limits.markets.proposeSpotEnabled"
	PerpsMarketTradingEnabled = "limits.markets.proposePerpetualEnabled"
	AMMMarketTradingEnabled   = "limits.markets.ammPoolEnabled"
	EthereumOraclesEnabled    = "ethereum.oracles.enabled"

	MarketMarginScalingFactors        = "market.margin.scalingFactors"
	MarketFeeFactorsMakerFee          = "market.fee.factors.makerFee"
	MarketFeeFactorsInfrastructureFee = "market.fee.factors.infrastructureFee"
	MarketAuctionMinimumDuration      = "market.auction.minimumDuration"
	MarketAuctionMaximumDuration      = "market.auction.maximumDuration"

	MarketTargetStakeTimeWindow               = "market.stake.target.timeWindow"
	MarketTargetStakeScalingFactor            = "market.stake.target.scalingFactor"
	MarketLiquidityTargetStakeTriggeringRatio = "market.liquidity.targetstake.triggering.ratio"
	MarketValueWindowLength                   = "market.value.windowLength"
	MarketPriceMonitoringDefaultParameters    = "market.monitor.price.defaultParameters"

	MarketMinLpStakeQuantumMultiple          = "market.liquidityProvision.minLpStakeQuantumMultiple"
	MarketProbabilityOfTradingTauScaling     = "market.liquidity.probabilityOfTrading.tau.scaling"
	MarketMinProbabilityOfTradingForLPOrders = "market.liquidity.minimum.probabilityOfTrading.lpOrders"
	MarketSuccessorLaunchWindow              = "market.liquidity.successorLaunchWindowLength"

	MarketLiquidityProvisionShapesMaxSize            = "market.liquidityProvision.shapes.maxSize"
	MarketLiquidityTargetStakeTriggeringRatioXXX     = "market.liquidity.targetstake.triggering.ratio"
	MarketLiquidityBondPenaltyParameter              = "market.liquidity.bondPenaltyParameter"
	MarketLiquidityEarlyExitPenalty                  = "market.liquidity.earlyExitPenalty"
	MarketLiquidityMaximumLiquidityFeeFactorLevel    = "market.liquidity.maximumLiquidityFeeFactorLevel"
	MarketLiquiditySLANonPerformanceBondPenaltyMax   = "market.liquidity.sla.nonPerformanceBondPenaltyMax"
	MarketLiquiditySLANonPerformanceBondPenaltySlope = "market.liquidity.sla.nonPerformanceBondPenaltySlope"
	MarketLiquidityStakeToCCYVolume                  = "market.liquidity.stakeToCcyVolume"
	MarketLiquidityProvidersFeeCalculationTimeStep   = "market.liquidity.providersFeeCalculationTimeStep"
	MarketLiquidityEquityLikeShareFeeFraction        = "market.liquidity.equityLikeShareFeeFraction"

	RewardAsset = "reward.asset"

	// market proposal parameters.
	GovernanceProposalMarketMinClose              = "governance.proposal.market.minClose"
	GovernanceProposalMarketMaxClose              = "governance.proposal.market.maxClose"
	GovernanceProposalMarketMinEnact              = "governance.proposal.market.minEnact"
	GovernanceProposalMarketMaxEnact              = "governance.proposal.market.maxEnact"
	GovernanceProposalMarketRequiredParticipation = "governance.proposal.market.requiredParticipation"
	GovernanceProposalMarketRequiredMajority      = "governance.proposal.market.requiredMajority"
	GovernanceProposalMarketMinProposerBalance    = "governance.proposal.market.minProposerBalance"
	GovernanceProposalMarketMinVoterBalance       = "governance.proposal.market.minVoterBalance"

	// asset proposal parameters.
	GovernanceProposalAssetMinClose              = "governance.proposal.asset.minClose"
	GovernanceProposalAssetMaxClose              = "governance.proposal.asset.maxClose"
	GovernanceProposalAssetMinEnact              = "governance.proposal.asset.minEnact"
	GovernanceProposalAssetMaxEnact              = "governance.proposal.asset.maxEnact"
	GovernanceProposalAssetRequiredParticipation = "governance.proposal.asset.requiredParticipation"
	GovernanceProposalAssetRequiredMajority      = "governance.proposal.asset.requiredMajority"
	GovernanceProposalAssetMinProposerBalance    = "governance.proposal.asset.minProposerBalance"
	GovernanceProposalAssetMinVoterBalance       = "governance.proposal.asset.minVoterBalance"

	// update asset proposal parameters.
	GovernanceProposalUpdateAssetMinClose              = "governance.proposal.updateAsset.minClose"
	GovernanceProposalUpdateAssetMaxClose              = "governance.proposal.updateAsset.maxClose"
	GovernanceProposalUpdateAssetMinEnact              = "governance.proposal.updateAsset.minEnact"
	GovernanceProposalUpdateAssetMaxEnact              = "governance.proposal.updateAsset.maxEnact"
	GovernanceProposalUpdateAssetRequiredParticipation = "governance.proposal.updateAsset.requiredParticipation"
	GovernanceProposalUpdateAssetRequiredMajority      = "governance.proposal.updateAsset.requiredMajority"
	GovernanceProposalUpdateAssetMinProposerBalance    = "governance.proposal.updateAsset.minProposerBalance"
	GovernanceProposalUpdateAssetMinVoterBalance       = "governance.proposal.updateAsset.minVoterBalance"

	// updateMarket proposal parameters.
	GovernanceProposalUpdateMarketMinClose                   = "governance.proposal.updateMarket.minClose"
	GovernanceProposalUpdateMarketMaxClose                   = "governance.proposal.updateMarket.maxClose"
	GovernanceProposalUpdateMarketMinEnact                   = "governance.proposal.updateMarket.minEnact"
	GovernanceProposalUpdateMarketMaxEnact                   = "governance.proposal.updateMarket.maxEnact"
	GovernanceProposalUpdateMarketRequiredParticipation      = "governance.proposal.updateMarket.requiredParticipation"
	GovernanceProposalUpdateMarketRequiredMajority           = "governance.proposal.updateMarket.requiredMajority"
	GovernanceProposalUpdateMarketMinProposerBalance         = "governance.proposal.updateMarket.minProposerBalance"
	GovernanceProposalUpdateMarketMinVoterBalance            = "governance.proposal.updateMarket.minVoterBalance"
	GovernanceProposalUpdateMarketMinProposerEquityLikeShare = "governance.proposal.updateMarket.minProposerEquityLikeShare"
	GovernanceProposalUpdateMarketRequiredParticipationLP    = "governance.proposal.updateMarket.requiredParticipationLP"
	GovernanceProposalUpdateMarketRequiredMajorityLP         = "governance.proposal.updateMarket.requiredMajorityLP"

	// updateNetParam proposal parameters.
	GovernanceProposalUpdateNetParamMinClose              = "governance.proposal.updateNetParam.minClose"
	GovernanceProposalUpdateNetParamMaxClose              = "governance.proposal.updateNetParam.maxClose"
	GovernanceProposalUpdateNetParamMinEnact              = "governance.proposal.updateNetParam.minEnact"
	GovernanceProposalUpdateNetParamMaxEnact              = "governance.proposal.updateNetParam.maxEnact"
	GovernanceProposalUpdateNetParamRequiredParticipation = "governance.proposal.updateNetParam.requiredParticipation"
	GovernanceProposalUpdateNetParamRequiredMajority      = "governance.proposal.updateNetParam.requiredMajority"
	GovernanceProposalUpdateNetParamMinProposerBalance    = "governance.proposal.updateNetParam.minProposerBalance"
	GovernanceProposalUpdateNetParamMinVoterBalance       = "governance.proposal.updateNetParam.minVoterBalance"

	// freeform proposal parameters.
	GovernanceProposalFreeformMinClose              = "governance.proposal.freeform.minClose"
	GovernanceProposalFreeformMaxClose              = "governance.proposal.freeform.maxClose"
	GovernanceProposalFreeformRequiredParticipation = "governance.proposal.freeform.requiredParticipation"
	GovernanceProposalFreeformRequiredMajority      = "governance.proposal.freeform.requiredMajority"
	GovernanceProposalFreeformMinProposerBalance    = "governance.proposal.freeform.minProposerBalance"
	GovernanceProposalFreeformMinVoterBalance       = "governance.proposal.freeform.minVoterBalance"

	// governance transfer proposal parameters.
	GovernanceProposalTransferMinClose              = "governance.proposal.transfer.minClose"
	GovernanceProposalTransferMaxClose              = "governance.proposal.transfer.maxClose"
	GovernanceProposalTransferMinEnact              = "governance.proposal.transfer.minEnact"
	GovernanceProposalTransferMaxEnact              = "governance.proposal.transfer.maxEnact"
	GovernanceProposalTransferRequiredParticipation = "governance.proposal.transfer.requiredParticipation"
	GovernanceProposalTransferRequiredMajority      = "governance.proposal.transfer.requiredMajority"
	GovernanceProposalTransferMinProposerBalance    = "governance.proposal.transfer.minProposerBalance"
	GovernanceProposalTransferMinVoterBalance       = "governance.proposal.transfer.minVoterBalance"
	GovernanceTransferMaxAmount                     = "governance.proposal.transfer.maxAmount"
	GovernanceTransferMaxFraction                   = "governance.proposal.transfer.maxFraction"

	// Network parameters for referral program update.
	GovernanceProposalReferralProgramMinClose              = "governance.proposal.referralProgram.minClose"
	GovernanceProposalReferralProgramMaxClose              = "governance.proposal.referralProgram.maxClose"
	GovernanceProposalReferralProgramMinEnact              = "governance.proposal.referralProgram.minEnact"
	GovernanceProposalReferralProgramMaxEnact              = "governance.proposal.referralProgram.maxEnact"
	GovernanceProposalReferralProgramRequiredParticipation = "governance.proposal.referralProgram.requiredParticipation"
	GovernanceProposalReferralProgramRequiredMajority      = "governance.proposal.referralProgram.requiredMajority"
	GovernanceProposalReferralProgramMinProposerBalance    = "governance.proposal.referralProgram.minProposerBalance"
	GovernanceProposalReferralProgramMinVoterBalance       = "governance.proposal.referralProgram.minVoterBalance"

	// Network parameters for referral program update.
	GovernanceProposalVolumeDiscountProgramMinClose              = "governance.proposal.VolumeDiscountProgram.minClose"
	GovernanceProposalVolumeDiscountProgramMaxClose              = "governance.proposal.VolumeDiscountProgram.maxClose"
	GovernanceProposalVolumeDiscountProgramMinEnact              = "governance.proposal.VolumeDiscountProgram.minEnact"
	GovernanceProposalVolumeDiscountProgramMaxEnact              = "governance.proposal.VolumeDiscountProgram.maxEnact"
	GovernanceProposalVolumeDiscountProgramRequiredParticipation = "governance.proposal.VolumeDiscountProgram.requiredParticipation"
	GovernanceProposalVolumeDiscountProgramRequiredMajority      = "governance.proposal.VolumeDiscountProgram.requiredMajority"
	GovernanceProposalVolumeDiscountProgramMinProposerBalance    = "governance.proposal.VolumeDiscountProgram.minProposerBalance"
	GovernanceProposalVolumeDiscountProgramMinVoterBalance       = "governance.proposal.VolumeDiscountProgram.minVoterBalance"

	// staking and delegation reward network params.
	StakingAndDelegationRewardPayoutFraction          = "reward.staking.delegation.payoutFraction"
	StakingAndDelegationRewardMaxPayoutPerParticipant = "reward.staking.delegation.maxPayoutPerParticipant"
	StakingAndDelegationRewardPayoutDelay             = "reward.staking.delegation.payoutDelay"
	StakingAndDelegationRewardDelegatorShare          = "reward.staking.delegation.delegatorShare"
	StakingAndDelegationRewardMinimumValidatorStake   = "reward.staking.delegation.minimumValidatorStake"
	StakingAndDelegationRewardCompetitionLevel        = "reward.staking.delegation.competitionLevel"
	StakingAndDelegationRewardMaxPayoutPerEpoch       = "reward.staking.delegation.maxPayoutPerEpoch"
	StakingAndDelegationRewardsMinValidators          = "reward.staking.delegation.minValidators"
	StakingAndDelegationRewardOptimalStakeMultiplier  = "reward.staking.delegation.optimalStakeMultiplier"

	RewardMarketCreationQuantumMultiple       = "rewards.marketCreationQuantumMultiple"
	MinEpochsInTeamForMetricRewardEligibility = "rewards.team.minEpochsInTeam"

	// spam policies params.
	SpamProtectionMaxVotes                         = "spam.protection.max.votes"
	SpamProtectionMinTokensForVoting               = "spam.protection.voting.min.tokens"
	SpamProtectionMaxProposals                     = "spam.protection.max.proposals"
	SpamProtectionMinTokensForProposal             = "spam.protection.proposal.min.tokens"
	SpamProtectionMaxDelegations                   = "spam.protection.max.delegations"
	SpamProtectionMinTokensForDelegation           = "spam.protection.delegation.min.tokens"
	SpamProtectionMaxBatchSize                     = "spam.protection.max.batchSize"
	SpamProtectionMinimumWithdrawalQuantumMultiple = "spam.protection.minimumWithdrawalQuantumMultiple"
	SpamProtectionMinMultisigUpdates               = "spam.protection.minMultisigUpdates"
	SpamProtectionMaxStopOrdersPerMarket           = "spam.protection.max.stopOrdersPerMarket"

	SpamProtectionMaxCreateReferralSet     = "spam.protection.max.createReferralSet"
	SpamProtectionMaxUpdateReferralSet     = "spam.protection.max.updateReferralSet"
	SpamProtectionMaxApplyReferralCode     = "spam.protection.max.applyReferralCode"
	SpamProtectionBalanceSnapshotFrequency = "spam.protection.balanceSnapshotFrequency"
	SpamProtectionApplyReferralMinFunds    = "spam.protection.applyReferral.min.funds"
	SpamProtectionReferralSetMinFunds      = "spam.protection.referralSet.min.funds"

	SpamProtectionMaxUpdatePartyProfile = "spam.protection.max.updatePartyProfile"
	SpamProtectionUpdateProfileMinFunds = "spam.protection.updatePartyProfile.min.funds"

	// blockchain specifics?
	BlockchainsEthereumConfig    = "blockchains.ethereumConfig"
	BlockchainsEthereumL2Configs = "blockchains.ethereumRpcAndEvmCompatDataSourcesConfig"

	// length of epoch in seconds.
	ValidatorsEpochLength = "validators.epoch.length"
	// delegation params.
	DelegationMinAmount = "validators.delegation.minAmount"

	ValidatorsVoteRequired = "validators.vote.required"

	// network related parameters.
	NetworkCheckpointTimeElapsedBetweenCheckpoints = "network.checkpoint.timeElapsedBetweenCheckpoints"

	// snapshot parameters.
	SnapshotIntervalLength = "snapshot.interval.length"

	FloatingPointUpdatesDuration = "network.floatingPointUpdates.delay"

	// validators by stake.
	NumberOfTendermintValidators               = "network.validators.tendermint.number"
	ValidatorIncumbentBonus                    = "network.validators.incumbentBonus"
	NumberEthMultisigSigners                   = "network.validators.multisig.numberOfSigners"
	ErsatzvalidatorsRewardFactor               = "network.validators.ersatz.rewardFactor"
	MultipleOfTendermintValidatorsForEtsatzSet = "network.validators.ersatz.multipleOfTendermintValidators"
	MinimumEthereumEventsForNewValidator       = "network.validators.minimumEthereumEventsForNewValidator"

	TransferFeeFactor                       = "transfer.fee.factor"
	TransferMinTransferQuantumMultiple      = "transfer.minTransferQuantumMultiple"
	TransferMaxCommandsPerEpoch             = "spam.protection.maxUserTransfersPerEpoch"
	TransferFeeMaxQuantumAmount             = "transfer.fee.maxQuantumAmount"
	TransferFeeDiscountDecayFraction        = "transfer.feeDiscountDecayFraction"
	TransferFeeDiscountMinimumTrackedAmount = "transfer.feeDiscountMinimumTrackedAmount"

	// proof of work.
	SpamPoWNumberOfPastBlocks   = "spam.pow.numberOfPastBlocks"
	SpamPoWDifficulty           = "spam.pow.difficulty"
	SpamPoWHashFunction         = "spam.pow.hashFunction"
	SpamPoWNumberOfTxPerBlock   = "spam.pow.numberOfTxPerBlock"
	SpamPoWIncreasingDifficulty = "spam.pow.increaseDifficulty"

	// limits.
	LimitsProposeMarketEnabledFrom = "limits.markets.proposeEnabledFrom"
	LimitsProposeAssetEnabledFrom  = "limits.assets.proposeEnabledFrom"

	MaxGasPerBlock   = "network.transactions.maxgasperblock"
	DefaultGas       = "network.transaction.defaultgas"
	MinBlockCapacity = "network.transactions.minBlockCapacity"

	// network wide limits.
	MaxPeggedOrders = "limits.markets.maxPeggedOrders"
	// MTM interval.
	MarkPriceUpdateMaximumFrequency = "network.markPriceUpdateMaximumFrequency"
	// interval for updating internal composite price for funding payment in perps.
	InternalCompositePriceUpdateFrequency = "network.internalCompositePriceUpdateFrequency"

	// how much to scale the number of proposed blocks used for performance calculation.
	ValidatorPerformanceScalingFactor = "validator.performance.scaling.factor"

	RewardsVestingBaseRate        = "rewards.vesting.baseRate"
	RewardsVestingMinimumTransfer = "rewards.vesting.minimumTransfer"
	RewardsVestingBenefitTiers    = "rewards.vesting.benefitTiers"

	// Referral program.
	ReferralProgramMaxReferralTiers                        = "referralProgram.maxReferralTiers"
	ReferralProgramMaxReferralRewardFactor                 = "referralProgram.maxReferralRewardFactor"
	ReferralProgramMaxReferralDiscountFactor               = "referralProgram.maxReferralDiscountFactor"
	ReferralProgramMaxPartyNotionalVolumeByQuantumPerEpoch = "referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch"
	ReferralProgramMinStakedVegaTokens                     = "referralProgram.minStakedVegaTokens"
	ReferralProgramMaxReferralRewardProportion             = "referralProgram.maxReferralRewardProportion"

	// volume discount program.
	VolumeDiscountProgramMaxBenefitTiers         = "volumeDiscountProgram.maxBenefitTiers"
	VolumeDiscountProgramMaxVolumeDiscountFactor = "volumeDiscountProgram.maxVolumeDiscountFactor"

	RewardsActivityStreakBenefitTiers          = "rewards.activityStreak.benefitTiers"
	RewardsActivityStreakInactivityLimit       = "rewards.activityStreak.inactivityLimit"
	RewardsActivityStreakMinQuantumOpenVolume  = "rewards.activityStreak.minQuantumOpenVolume"
	RewardsActivityStreakMinQuantumTradeVolume = "rewards.activityStreak.minQuantumTradeVolume"

	MarketAMMMinCommitmentQuantum = "market.amm.minCommitmentQuantum"
)

var Deprecated = map[string]struct{}{
	StakingAndDelegationRewardPayoutFraction:    {},
	StakingAndDelegationRewardPayoutDelay:       {},
	StakingAndDelegationRewardMaxPayoutPerEpoch: {},
	MarketLiquidityTargetStakeTriggeringRatio:   {},
	MarketTargetStakeTimeWindow:                 {},
	MarketTargetStakeScalingFactor:              {},
}

var AllKeys = map[string]struct{}{
	SpamProtectionMaxUpdatePartyProfile:                          {},
	SpamProtectionUpdateProfileMinFunds:                          {},
	MarketAMMMinCommitmentQuantum:                                {},
	GovernanceProposalVolumeDiscountProgramMinClose:              {},
	GovernanceProposalVolumeDiscountProgramMaxClose:              {},
	GovernanceProposalVolumeDiscountProgramMinEnact:              {},
	GovernanceProposalVolumeDiscountProgramMaxEnact:              {},
	GovernanceProposalVolumeDiscountProgramRequiredParticipation: {},
	GovernanceProposalVolumeDiscountProgramRequiredMajority:      {},
	GovernanceProposalVolumeDiscountProgramMinProposerBalance:    {},
	GovernanceProposalVolumeDiscountProgramMinVoterBalance:       {},
	ReferralProgramMaxReferralRewardProportion:                   {},
	MinEpochsInTeamForMetricRewardEligibility:                    {},
	RewardsVestingBenefitTiers:                                   {},
	RewardsVestingMinimumTransfer:                                {},
	RewardsActivityStreakInactivityLimit:                         {},
	RewardsActivityStreakBenefitTiers:                            {},
	RewardsActivityStreakMinQuantumOpenVolume:                    {},
	RewardsActivityStreakMinQuantumTradeVolume:                   {},
	RewardsVestingBaseRate:                                       {},
	SpotMarketTradingEnabled:                                     {},
	PerpsMarketTradingEnabled:                                    {},
	AMMMarketTradingEnabled:                                      {},
	EthereumOraclesEnabled:                                       {},
	MaxPeggedOrders:                                              {},
	MaxGasPerBlock:                                               {},
	DefaultGas:                                                   {},
	MinBlockCapacity:                                             {},
	RewardMarketCreationQuantumMultiple:                          {},
	MarketMarginScalingFactors:                                   {},
	MarketFeeFactorsMakerFee:                                     {},
	MarketFeeFactorsInfrastructureFee:                            {},
	MarketAuctionMinimumDuration:                                 {},
	MarketAuctionMaximumDuration:                                 {},
	MarketLiquidityBondPenaltyParameter:                          {},
	MarketLiquidityMaximumLiquidityFeeFactorLevel:                {},
	MarketLiquidityTargetStakeTriggeringRatio:                    {},
	MarketLiquidityEarlyExitPenalty:                              {},
	MarketLiquiditySLANonPerformanceBondPenaltySlope:             {},
	MarketLiquiditySLANonPerformanceBondPenaltyMax:               {},
	MarketLiquidityStakeToCCYVolume:                              {},
	MarketLiquidityProvidersFeeCalculationTimeStep:               {},
	MarketTargetStakeTimeWindow:                                  {},
	MarketTargetStakeScalingFactor:                               {},
	MarketPriceMonitoringDefaultParameters:                       {},
	MarketMinLpStakeQuantumMultiple:                              {},
	RewardAsset:                                                  {},
	GovernanceProposalMarketMinClose:                             {},
	GovernanceProposalMarketMaxClose:                             {},
	GovernanceProposalMarketMinEnact:                             {},
	GovernanceProposalMarketMaxEnact:                             {},
	GovernanceProposalMarketRequiredParticipation:                {},
	GovernanceProposalMarketRequiredMajority:                     {},
	GovernanceProposalMarketMinProposerBalance:                   {},
	GovernanceProposalMarketMinVoterBalance:                      {},
	GovernanceProposalAssetMinClose:                              {},
	GovernanceProposalAssetMaxClose:                              {},
	GovernanceProposalAssetMinEnact:                              {},
	GovernanceProposalAssetMaxEnact:                              {},
	GovernanceProposalAssetRequiredParticipation:                 {},
	GovernanceProposalAssetRequiredMajority:                      {},
	GovernanceProposalAssetMinProposerBalance:                    {},
	GovernanceProposalAssetMinVoterBalance:                       {},
	GovernanceProposalUpdateMarketMinClose:                       {},
	GovernanceProposalUpdateMarketMaxClose:                       {},
	GovernanceProposalUpdateMarketMinEnact:                       {},
	GovernanceProposalUpdateMarketMaxEnact:                       {},
	GovernanceProposalUpdateMarketRequiredParticipation:          {},
	GovernanceProposalUpdateMarketRequiredMajority:               {},
	GovernanceProposalUpdateMarketMinProposerBalance:             {},
	GovernanceProposalUpdateMarketMinVoterBalance:                {},
	GovernanceProposalUpdateNetParamMinClose:                     {},
	GovernanceProposalUpdateNetParamMaxClose:                     {},
	GovernanceProposalUpdateNetParamMinEnact:                     {},
	GovernanceProposalUpdateNetParamMaxEnact:                     {},
	GovernanceProposalUpdateNetParamRequiredParticipation:        {},
	GovernanceProposalUpdateNetParamRequiredMajority:             {},
	GovernanceProposalUpdateNetParamMinProposerBalance:           {},
	GovernanceProposalUpdateNetParamMinVoterBalance:              {},
	GovernanceProposalUpdateMarketRequiredParticipationLP:        {},
	GovernanceProposalUpdateMarketRequiredMajorityLP:             {},
	GovernanceProposalUpdateMarketMinProposerEquityLikeShare:     {},
	GovernanceProposalFreeformMinClose:                           {},
	GovernanceProposalFreeformMaxClose:                           {},
	GovernanceProposalFreeformRequiredParticipation:              {},
	GovernanceProposalFreeformRequiredMajority:                   {},
	GovernanceProposalFreeformMinProposerBalance:                 {},
	GovernanceProposalFreeformMinVoterBalance:                    {},
	GovernanceProposalTransferMinClose:                           {},
	GovernanceProposalTransferMaxClose:                           {},
	GovernanceProposalTransferMinEnact:                           {},
	GovernanceProposalTransferMaxEnact:                           {},
	GovernanceProposalTransferRequiredParticipation:              {},
	GovernanceProposalTransferRequiredMajority:                   {},
	GovernanceProposalTransferMinProposerBalance:                 {},
	GovernanceProposalTransferMinVoterBalance:                    {},
	GovernanceTransferMaxAmount:                                  {},
	GovernanceTransferMaxFraction:                                {},
	GovernanceProposalReferralProgramMinClose:                    {},
	GovernanceProposalReferralProgramMaxClose:                    {},
	GovernanceProposalReferralProgramMinEnact:                    {},
	GovernanceProposalReferralProgramMaxEnact:                    {},
	GovernanceProposalReferralProgramRequiredParticipation:       {},
	GovernanceProposalReferralProgramRequiredMajority:            {},
	GovernanceProposalReferralProgramMinProposerBalance:          {},
	GovernanceProposalReferralProgramMinVoterBalance:             {},
	BlockchainsEthereumConfig:                                    {},
	MarketLiquidityProvisionShapesMaxSize:                        {},
	MarketProbabilityOfTradingTauScaling:                         {},
	MarketMinProbabilityOfTradingForLPOrders:                     {},
	ValidatorsEpochLength:                                        {},
	DelegationMinAmount:                                          {},
	StakingAndDelegationRewardPayoutFraction:                     {},
	StakingAndDelegationRewardMaxPayoutPerParticipant:            {},
	StakingAndDelegationRewardPayoutDelay:                        {},
	StakingAndDelegationRewardDelegatorShare:                     {},
	StakingAndDelegationRewardMinimumValidatorStake:              {},
	ValidatorsVoteRequired:                                       {},
	NetworkCheckpointTimeElapsedBetweenCheckpoints:               {},
	MarketValueWindowLength:                                      {},
	StakingAndDelegationRewardMaxPayoutPerEpoch:                  {},
	SpamProtectionMinTokensForProposal:                           {},
	SpamProtectionMaxVotes:                                       {},
	SpamProtectionMaxProposals:                                   {},
	SpamProtectionMinTokensForVoting:                             {},
	SpamProtectionMaxDelegations:                                 {},
	SpamProtectionMinTokensForDelegation:                         {},
	StakingAndDelegationRewardCompetitionLevel:                   {},
	StakingAndDelegationRewardsMinValidators:                     {},
	StakingAndDelegationRewardOptimalStakeMultiplier:             {},
	SnapshotIntervalLength:                                       {},
	FloatingPointUpdatesDuration:                                 {},
	TransferFeeFactor:                                            {},
	NumberOfTendermintValidators:                                 {},
	ValidatorIncumbentBonus:                                      {},
	NumberEthMultisigSigners:                                     {},
	ErsatzvalidatorsRewardFactor:                                 {},
	MultipleOfTendermintValidatorsForEtsatzSet:                   {},
	MinimumEthereumEventsForNewValidator:                         {},
	TransferMinTransferQuantumMultiple:                           {},
	TransferFeeMaxQuantumAmount:                                  {},
	TransferFeeDiscountDecayFraction:                             {},
	TransferFeeDiscountMinimumTrackedAmount:                      {},
	TransferMaxCommandsPerEpoch:                                  {},
	SpamPoWNumberOfPastBlocks:                                    {},
	SpamPoWDifficulty:                                            {},
	SpamPoWHashFunction:                                          {},
	SpamPoWNumberOfTxPerBlock:                                    {},
	SpamPoWIncreasingDifficulty:                                  {},
	LimitsProposeMarketEnabledFrom:                               {},
	LimitsProposeAssetEnabledFrom:                                {},
	GovernanceProposalUpdateAssetMinClose:                        {},
	GovernanceProposalUpdateAssetMaxClose:                        {},
	GovernanceProposalUpdateAssetMinEnact:                        {},
	GovernanceProposalUpdateAssetMaxEnact:                        {},
	GovernanceProposalUpdateAssetRequiredParticipation:           {},
	GovernanceProposalUpdateAssetRequiredMajority:                {},
	GovernanceProposalUpdateAssetMinProposerBalance:              {},
	GovernanceProposalUpdateAssetMinVoterBalance:                 {},
	SpamProtectionMaxBatchSize:                                   {},
	MarkPriceUpdateMaximumFrequency:                              {},
	InternalCompositePriceUpdateFrequency:                        {},
	ValidatorPerformanceScalingFactor:                            {},
	SpamProtectionMinimumWithdrawalQuantumMultiple:               {},
	SpamProtectionMinMultisigUpdates:                             {},
	MarketSuccessorLaunchWindow:                                  {},
	SpamProtectionMaxStopOrdersPerMarket:                         {},
	ReferralProgramMaxReferralTiers:                              {},
	ReferralProgramMaxReferralRewardFactor:                       {},
	ReferralProgramMaxReferralDiscountFactor:                     {},
	ReferralProgramMaxPartyNotionalVolumeByQuantumPerEpoch:       {},
	ReferralProgramMinStakedVegaTokens:                           {},
	VolumeDiscountProgramMaxBenefitTiers:                         {},
	VolumeDiscountProgramMaxVolumeDiscountFactor:                 {},
	SpamProtectionMaxCreateReferralSet:                           {},
	SpamProtectionMaxUpdateReferralSet:                           {},
	SpamProtectionMaxApplyReferralCode:                           {},
	SpamProtectionBalanceSnapshotFrequency:                       {},
	SpamProtectionApplyReferralMinFunds:                          {},
	SpamProtectionReferralSetMinFunds:                            {},
	BlockchainsEthereumL2Configs:                                 {},
}
