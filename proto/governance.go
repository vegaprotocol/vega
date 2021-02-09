package proto

func (l *NewMarketCommitment) IntoSubmission(
	market string) *LiquidityProvisionSubmission {
	return &LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: l.CommitmentAmount,
		Fee:              l.Fee,
		Sells:            l.Sells,
		Buys:             l.Buys,
	}
}
