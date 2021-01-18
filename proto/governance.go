package proto

func (l *NewMarketCommitment) IntoSubmission(
	market string) *LiquidityProvisionSubmission {
	return &LiquidityProvisionSubmission{
		MarketID:         market,
		CommitmentAmount: l.CommitmentAmount,
		Fee:              l.Fee,
		Sells:            l.Sells,
		Buys:             l.Buys,
	}
}
