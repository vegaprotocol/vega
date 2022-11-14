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

const (
	// market related parameters.
	MarketMarginScalingFactors                      = "market.margin.scalingFactors"
	MarketFeeFactorsMakerFee                        = "market.fee.factors.makerFee"
	MarketFeeFactorsInfrastructureFee               = "market.fee.factors.infrastructureFee"
	MarketAuctionMinimumDuration                    = "market.auction.minimumDuration"
	MarketAuctionMaximumDuration                    = "market.auction.maximumDuration"
	MarketLiquidityBondPenaltyParameter             = "market.liquidity.bondPenaltyParameter"
	MarketLiquidityMaximumLiquidityFeeFactorLevel   = "market.liquidity.maximumLiquidityFeeFactorLevel"
	MarketLiquidityStakeToCCYSiskas                 = "market.liquidity.stakeToCcySiskas"
	MarketLiquidityProvidersFeeDistribitionTimeStep = "market.liquidity.providers.fee.distributionTimeStep"
	MarketLiquidityTargetStakeTriggeringRatio       = "market.liquidity.targetstake.triggering.ratio"
	MarketProbabilityOfTradingTauScaling            = "market.liquidity.probabilityOfTrading.tau.scaling"
	MarketMinProbabilityOfTradingForLPOrders        = "market.liquidity.minimum.probabilityOfTrading.lpOrders"
	MarketTargetStakeTimeWindow                     = "market.stake.target.timeWindow"
	MarketTargetStakeScalingFactor                  = "market.stake.target.scalingFactor"
	MarketValueWindowLength                         = "market.value.windowLength"
	MarketPriceMonitoringDefaultParameters          = "market.monitor.price.defaultParameters"
	MarketLiquidityProvisionShapesMaxSize           = "market.liquidityProvision.shapes.maxSize"
	MarketMinLpStakeQuantumMultiple                 = "market.liquidityProvision.minLpStakeQuantumMultiple"

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

	RewardMarketCreationQuantumMultiple = "rewards.marketCreationQuantumMultiple"

	// spam policies params.
	SpamProtectionMaxVotes               = "spam.protection.max.votes"
	SpamProtectionMinTokensForVoting     = "spam.protection.voting.min.tokens"
	SpamProtectionMaxProposals           = "spam.protection.max.proposals"
	SpamProtectionMinTokensForProposal   = "spam.protection.proposal.min.tokens"
	SpamProtectionMaxDelegations         = "spam.protection.max.delegations"
	SpamProtectionMinTokensForDelegation = "spam.protection.delegation.min.tokens"
	SpamProtectionMaxBatchSize           = "spam.protection.max.batchSize"

	// blockchain specifics?
	BlockchainsEthereumConfig = "blockchains.ethereumConfig"

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

	TransferFeeFactor                  = "transfer.fee.factor"
	TransferMinTransferQuantumMultiple = "transfer.minTransferQuantumMultiple"
	TransferMaxCommandsPerEpoch        = "spam.protection.maxUserTransfersPerEpoch"

	// proof of work.
	SpamPoWNumberOfPastBlocks   = "spam.pow.numberOfPastBlocks"
	SpamPoWDifficulty           = "spam.pow.difficulty"
	SpamPoWHashFunction         = "spam.pow.hashFunction"
	SpamPoWNumberOfTxPerBlock   = "spam.pow.numberOfTxPerBlock"
	SpamPoWIncreasingDifficulty = "spam.pow.increaseDifficulty"

	// limits.
	LimitsProposeMarketEnabledFrom = "limits.markets.proposeEnabledFrom"
	LimitsProposeAssetEnabledFrom  = "limits.assets.proposeEnabledFrom"

	MaxGasPerBlock = "network.transactions.maxgasperblock"
	DefaultGas     = "network.transaction.defaultgas"

	// network wide limits.
	MaxPeggedOrders = "limits.markets.maxPeggedOrders"
	// MTM interval
	MarkPriceUpdateMaximumFrequency = "network.markPriceUpdateMaximumFrequency"
)

var AllKeys = map[string]struct{}{
	MaxPeggedOrders:                                          {},
	MaxGasPerBlock:                                           {},
	DefaultGas:                                               {},
	RewardMarketCreationQuantumMultiple:                      {},
	MarketMarginScalingFactors:                               {},
	MarketFeeFactorsMakerFee:                                 {},
	MarketFeeFactorsInfrastructureFee:                        {},
	MarketAuctionMinimumDuration:                             {},
	MarketAuctionMaximumDuration:                             {},
	MarketLiquidityBondPenaltyParameter:                      {},
	MarketLiquidityMaximumLiquidityFeeFactorLevel:            {},
	MarketLiquidityStakeToCCYSiskas:                          {},
	MarketLiquidityProvidersFeeDistribitionTimeStep:          {},
	MarketLiquidityTargetStakeTriggeringRatio:                {},
	MarketTargetStakeTimeWindow:                              {},
	MarketTargetStakeScalingFactor:                           {},
	MarketPriceMonitoringDefaultParameters:                   {},
	MarketMinLpStakeQuantumMultiple:                          {},
	RewardAsset:                                              {},
	GovernanceProposalMarketMinClose:                         {},
	GovernanceProposalMarketMaxClose:                         {},
	GovernanceProposalMarketMinEnact:                         {},
	GovernanceProposalMarketMaxEnact:                         {},
	GovernanceProposalMarketRequiredParticipation:            {},
	GovernanceProposalMarketRequiredMajority:                 {},
	GovernanceProposalMarketMinProposerBalance:               {},
	GovernanceProposalMarketMinVoterBalance:                  {},
	GovernanceProposalAssetMinClose:                          {},
	GovernanceProposalAssetMaxClose:                          {},
	GovernanceProposalAssetMinEnact:                          {},
	GovernanceProposalAssetMaxEnact:                          {},
	GovernanceProposalAssetRequiredParticipation:             {},
	GovernanceProposalAssetRequiredMajority:                  {},
	GovernanceProposalAssetMinProposerBalance:                {},
	GovernanceProposalAssetMinVoterBalance:                   {},
	GovernanceProposalUpdateMarketMinClose:                   {},
	GovernanceProposalUpdateMarketMaxClose:                   {},
	GovernanceProposalUpdateMarketMinEnact:                   {},
	GovernanceProposalUpdateMarketMaxEnact:                   {},
	GovernanceProposalUpdateMarketRequiredParticipation:      {},
	GovernanceProposalUpdateMarketRequiredMajority:           {},
	GovernanceProposalUpdateMarketMinProposerBalance:         {},
	GovernanceProposalUpdateMarketMinVoterBalance:            {},
	GovernanceProposalUpdateNetParamMinClose:                 {},
	GovernanceProposalUpdateNetParamMaxClose:                 {},
	GovernanceProposalUpdateNetParamMinEnact:                 {},
	GovernanceProposalUpdateNetParamMaxEnact:                 {},
	GovernanceProposalUpdateNetParamRequiredParticipation:    {},
	GovernanceProposalUpdateNetParamRequiredMajority:         {},
	GovernanceProposalUpdateNetParamMinProposerBalance:       {},
	GovernanceProposalUpdateNetParamMinVoterBalance:          {},
	GovernanceProposalUpdateMarketRequiredParticipationLP:    {},
	GovernanceProposalUpdateMarketRequiredMajorityLP:         {},
	GovernanceProposalUpdateMarketMinProposerEquityLikeShare: {},
	GovernanceProposalFreeformMinClose:                       {},
	GovernanceProposalFreeformMaxClose:                       {},
	GovernanceProposalFreeformRequiredParticipation:          {},
	GovernanceProposalFreeformRequiredMajority:               {},
	GovernanceProposalFreeformMinProposerBalance:             {},
	GovernanceProposalFreeformMinVoterBalance:                {},
	BlockchainsEthereumConfig:                                {},
	MarketLiquidityProvisionShapesMaxSize:                    {},
	MarketProbabilityOfTradingTauScaling:                     {},
	MarketMinProbabilityOfTradingForLPOrders:                 {},
	ValidatorsEpochLength:                                    {},
	DelegationMinAmount:                                      {},
	StakingAndDelegationRewardPayoutFraction:                 {},
	StakingAndDelegationRewardMaxPayoutPerParticipant:        {},
	StakingAndDelegationRewardPayoutDelay:                    {},
	StakingAndDelegationRewardDelegatorShare:                 {},
	StakingAndDelegationRewardMinimumValidatorStake:          {},
	ValidatorsVoteRequired:                                   {},
	NetworkCheckpointTimeElapsedBetweenCheckpoints:           {},
	MarketValueWindowLength:                                  {},
	StakingAndDelegationRewardMaxPayoutPerEpoch:              {},
	SpamProtectionMinTokensForProposal:                       {},
	SpamProtectionMaxVotes:                                   {},
	SpamProtectionMaxProposals:                               {},
	SpamProtectionMinTokensForVoting:                         {},
	SpamProtectionMaxDelegations:                             {},
	SpamProtectionMinTokensForDelegation:                     {},
	StakingAndDelegationRewardCompetitionLevel:               {},
	StakingAndDelegationRewardsMinValidators:                 {},
	StakingAndDelegationRewardOptimalStakeMultiplier:         {},
	SnapshotIntervalLength:                                   {},
	FloatingPointUpdatesDuration:                             {},
	TransferFeeFactor:                                        {},
	NumberOfTendermintValidators:                             {},
	ValidatorIncumbentBonus:                                  {},
	NumberEthMultisigSigners:                                 {},
	ErsatzvalidatorsRewardFactor:                             {},
	MultipleOfTendermintValidatorsForEtsatzSet:               {},
	MinimumEthereumEventsForNewValidator:                     {},
	TransferMinTransferQuantumMultiple:                       {},
	TransferMaxCommandsPerEpoch:                              {},
	SpamPoWNumberOfPastBlocks:                                {},
	SpamPoWDifficulty:                                        {},
	SpamPoWHashFunction:                                      {},
	SpamPoWNumberOfTxPerBlock:                                {},
	SpamPoWIncreasingDifficulty:                              {},
	LimitsProposeMarketEnabledFrom:                           {},
	LimitsProposeAssetEnabledFrom:                            {},
	GovernanceProposalUpdateAssetMinClose:                    {},
	GovernanceProposalUpdateAssetMaxClose:                    {},
	GovernanceProposalUpdateAssetMinEnact:                    {},
	GovernanceProposalUpdateAssetMaxEnact:                    {},
	GovernanceProposalUpdateAssetRequiredParticipation:       {},
	GovernanceProposalUpdateAssetRequiredMajority:            {},
	GovernanceProposalUpdateAssetMinProposerBalance:          {},
	GovernanceProposalUpdateAssetMinVoterBalance:             {},
	SpamProtectionMaxBatchSize:                               {},
}
