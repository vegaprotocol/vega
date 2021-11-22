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
	MarketPriceMonitoringUpdateFrequency            = "market.monitor.price.updateFrequency"
	MarketLiquidityProvisionShapesMaxSize           = "market.liquidityProvision.shapes.maxSize"

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

	// updateMarket proposal parameters.
	GovernanceProposalUpdateMarketMinClose              = "governance.proposal.updateMarket.minClose"
	GovernanceProposalUpdateMarketMaxClose              = "governance.proposal.updateMarket.maxClose"
	GovernanceProposalUpdateMarketMinEnact              = "governance.proposal.updateMarket.minEnact"
	GovernanceProposalUpdateMarketMaxEnact              = "governance.proposal.updateMarket.maxEnact"
	GovernanceProposalUpdateMarketRequiredParticipation = "governance.proposal.updateMarket.requiredParticipation"
	GovernanceProposalUpdateMarketRequiredMajority      = "governance.proposal.updateMarket.requiredMajority"
	GovernanceProposalUpdateMarketMinProposerBalance    = "governance.proposal.updateMarket.minProposerBalance"
	GovernanceProposalUpdateMarketMinVoterBalance       = "governance.proposal.updateMarket.minVoterBalance"

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

	// spam policies params.
	SpamProtectionMaxVotes               = "spam.protection.max.votes"
	SpamProtectionMinTokensForVoting     = "spam.protection.voting.min.tokens"
	SpamProtectionMaxProposals           = "spam.protection.max.proposals"
	SpamProtectionMinTokensForProposal   = "spam.protection.proposal.min.tokens"
	SpamProtectionMaxDelegations         = "spam.protection.max.delegations"
	SpamProtectionMinTokensForDelegation = "spam.protection.delegation.min.tokens"

	// blockchain specifics?
	BlockchainsEthereumConfig = "blockchains.ethereumConfig"

	// length of epoch in seconds.
	ValidatorsEpochLength = "validators.epoch.length"
	// delegation params.
	DelegationMinAmount = "validators.delegation.minAmount"

	ValidatorsVoteRequired = "validators.vote.required"

	// network related parameters.
	NetworkCheckpointMarketFreezeDate              = "network.checkpoint.marketFreezeDate"
	NetworkCheckpointNetworkEOLDate                = "network.checkpoint.networkEndOfLifeDate"
	NetworkCheckpointTimeElapsedBetweenCheckpoints = "network.checkpoint.timeElapsedBetweenCheckpoints"

	// snapshot parameters.
	SnapshotIntervalLength = "snapshot.interval.length"
)

var AllKeys = map[string]struct{}{
	MarketMarginScalingFactors:                            {},
	MarketFeeFactorsMakerFee:                              {},
	MarketFeeFactorsInfrastructureFee:                     {},
	MarketAuctionMinimumDuration:                          {},
	MarketAuctionMaximumDuration:                          {},
	MarketLiquidityBondPenaltyParameter:                   {},
	MarketLiquidityMaximumLiquidityFeeFactorLevel:         {},
	MarketLiquidityStakeToCCYSiskas:                       {},
	MarketLiquidityProvidersFeeDistribitionTimeStep:       {},
	MarketLiquidityTargetStakeTriggeringRatio:             {},
	MarketTargetStakeTimeWindow:                           {},
	MarketTargetStakeScalingFactor:                        {},
	MarketPriceMonitoringDefaultParameters:                {},
	MarketPriceMonitoringUpdateFrequency:                  {},
	RewardAsset:                                           {},
	GovernanceProposalMarketMinClose:                      {},
	GovernanceProposalMarketMaxClose:                      {},
	GovernanceProposalMarketMinEnact:                      {},
	GovernanceProposalMarketMaxEnact:                      {},
	GovernanceProposalMarketRequiredParticipation:         {},
	GovernanceProposalMarketRequiredMajority:              {},
	GovernanceProposalMarketMinProposerBalance:            {},
	GovernanceProposalMarketMinVoterBalance:               {},
	GovernanceProposalAssetMinClose:                       {},
	GovernanceProposalAssetMaxClose:                       {},
	GovernanceProposalAssetMinEnact:                       {},
	GovernanceProposalAssetMaxEnact:                       {},
	GovernanceProposalAssetRequiredParticipation:          {},
	GovernanceProposalAssetRequiredMajority:               {},
	GovernanceProposalAssetMinProposerBalance:             {},
	GovernanceProposalAssetMinVoterBalance:                {},
	GovernanceProposalUpdateMarketMinClose:                {},
	GovernanceProposalUpdateMarketMaxClose:                {},
	GovernanceProposalUpdateMarketMinEnact:                {},
	GovernanceProposalUpdateMarketMaxEnact:                {},
	GovernanceProposalUpdateMarketRequiredParticipation:   {},
	GovernanceProposalUpdateMarketRequiredMajority:        {},
	GovernanceProposalUpdateMarketMinProposerBalance:      {},
	GovernanceProposalUpdateMarketMinVoterBalance:         {},
	GovernanceProposalUpdateNetParamMinClose:              {},
	GovernanceProposalUpdateNetParamMaxClose:              {},
	GovernanceProposalUpdateNetParamMinEnact:              {},
	GovernanceProposalUpdateNetParamMaxEnact:              {},
	GovernanceProposalUpdateNetParamRequiredParticipation: {},
	GovernanceProposalUpdateNetParamRequiredMajority:      {},
	GovernanceProposalUpdateNetParamMinProposerBalance:    {},
	GovernanceProposalUpdateNetParamMinVoterBalance:       {},
	BlockchainsEthereumConfig:                             {},
	MarketLiquidityProvisionShapesMaxSize:                 {},
	MarketProbabilityOfTradingTauScaling:                  {},
	MarketMinProbabilityOfTradingForLPOrders:              {},
	ValidatorsEpochLength:                                 {},
	DelegationMinAmount:                                   {},
	StakingAndDelegationRewardPayoutFraction:              {},
	StakingAndDelegationRewardMaxPayoutPerParticipant:     {},
	StakingAndDelegationRewardPayoutDelay:                 {},
	StakingAndDelegationRewardDelegatorShare:              {},
	StakingAndDelegationRewardMinimumValidatorStake:       {},
	ValidatorsVoteRequired:                                {},
	NetworkCheckpointNetworkEOLDate:                       {},
	NetworkCheckpointTimeElapsedBetweenCheckpoints:        {},
	NetworkCheckpointMarketFreezeDate:                     {},
	MarketValueWindowLength:                               {},
	StakingAndDelegationRewardMaxPayoutPerEpoch:           {},
	SpamProtectionMinTokensForProposal:                    {},
	SpamProtectionMaxVotes:                                {},
	SpamProtectionMaxProposals:                            {},
	SpamProtectionMinTokensForVoting:                      {},
	SpamProtectionMaxDelegations:                          {},
	SpamProtectionMinTokensForDelegation:                  {},
	StakingAndDelegationRewardCompetitionLevel:            {},
	StakingAndDelegationRewardsMinValidators:              {},
	StakingAndDelegationRewardOptimalStakeMultiplier:      {},
	SnapshotIntervalLength:                                {},
}
