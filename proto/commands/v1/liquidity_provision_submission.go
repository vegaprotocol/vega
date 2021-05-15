package v1

import (
	types "code.vegaprotocol.io/vega/proto"
)

func LiquidityProvisionSubmissionFromMarketCommitment(
	nmc *types.NewMarketCommitment,
	market string,
) *LiquidityProvisionSubmission {
	return &LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: nmc.CommitmentAmount,
		Fee:              nmc.Fee,
		Sells:            nmc.Sells,
		Buys:             nmc.Buys,
		Reference:        nmc.Reference,
	}
}
