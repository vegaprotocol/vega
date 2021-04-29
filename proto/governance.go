package proto

func (n *NewMarketCommitment) IntoSubmission(
	market string) *LiquidityProvisionSubmission {
	return &LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: n.CommitmentAmount,
		Fee:              n.Fee,
		Sells:            n.Sells,
		Buys:             n.Buys,
		Reference:        n.Reference,
	}
}

func (p *Proposal) IntoSubmission() *ProposalSubmission {
	return &ProposalSubmission{
		Reference: p.Reference,
		Terms:     p.Terms,
	}
}
