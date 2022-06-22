// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package netparams

const (
	// market related parameters
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

	GovernanceVoteAsset = "governance.vote.asset"

	// market proposal parameters
	GovernanceProposalMarketMinClose              = "governance.proposal.market.minClose"
	GovernanceProposalMarketMaxClose              = "governance.proposal.market.maxClose"
	GovernanceProposalMarketMinEnact              = "governance.proposal.market.minEnact"
	GovernanceProposalMarketMaxEnact              = "governance.proposal.market.maxEnact"
	GovernanceProposalMarketRequiredParticipation = "governance.proposal.market.requiredParticipation"
	GovernanceProposalMarketRequiredMajority      = "governance.proposal.market.requiredMajority"
	GovernanceProposalMarketMinProposerBalance    = "governance.proposal.market.minProposerBalance"
	GovernanceProposalMarketMinVoterBalance       = "governance.proposal.market.minVoterBalance"

	// asset proposal parameters
	GovernanceProposalAssetMinClose              = "governance.proposal.asset.minClose"
	GovernanceProposalAssetMaxClose              = "governance.proposal.asset.maxClose"
	GovernanceProposalAssetMinEnact              = "governance.proposal.asset.minEnact"
	GovernanceProposalAssetMaxEnact              = "governance.proposal.asset.maxEnact"
	GovernanceProposalAssetRequiredParticipation = "governance.proposal.asset.requiredParticipation"
	GovernanceProposalAssetRequiredMajority      = "governance.proposal.asset.requiredMajority"
	GovernanceProposalAssetMinProposerBalance    = "governance.proposal.asset.minProposerBalance"
	GovernanceProposalAssetMinVoterBalance       = "governance.proposal.asset.minVoterBalance"

	// updateMarket proposal parameters
	GovernanceProposalUpdateMarketMinClose              = "governance.proposal.updateMarket.minClose"
	GovernanceProposalUpdateMarketMaxClose              = "governance.proposal.updateMarket.maxClose"
	GovernanceProposalUpdateMarketMinEnact              = "governance.proposal.updateMarket.minEnact"
	GovernanceProposalUpdateMarketMaxEnact              = "governance.proposal.updateMarket.maxEnact"
	GovernanceProposalUpdateMarketRequiredParticipation = "governance.proposal.updateMarket.requiredParticipation"
	GovernanceProposalUpdateMarketRequiredMajority      = "governance.proposal.updateMarket.requiredMajority"
	GovernanceProposalUpdateMarketMinProposerBalance    = "governance.proposal.updateMarket.minProposerBalance"
	GovernanceProposalUpdateMarketMinVoterBalance       = "governance.proposal.updateMarket.minVoterBalance"

	// updateNetParam proposal parameters
	GovernanceProposalUpdateNetParamMinClose              = "governance.proposal.updateNetParam.minClose"
	GovernanceProposalUpdateNetParamMaxClose              = "governance.proposal.updateNetParam.maxClose"
	GovernanceProposalUpdateNetParamMinEnact              = "governance.proposal.updateNetParam.minEnact"
	GovernanceProposalUpdateNetParamMaxEnact              = "governance.proposal.updateNetParam.maxEnact"
	GovernanceProposalUpdateNetParamRequiredParticipation = "governance.proposal.updateNetParam.requiredParticipation"
	GovernanceProposalUpdateNetParamRequiredMajority      = "governance.proposal.updateNetParam.requiredMajority"
	GovernanceProposalUpdateNetParamMinProposerBalance    = "governance.proposal.updateNetParam.minProposerBalance"
	GovernanceProposalUpdateNetParamMinVoterBalance       = "governance.proposal.updateNetParam.minVoterBalance"
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
	GovernanceVoteAsset:                                   {},
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
	MarketLiquidityProvisionShapesMaxSize:                 {},
	MarketProbabilityOfTradingTauScaling:                  {},
	MarketMinProbabilityOfTradingForLPOrders:              {},
}
