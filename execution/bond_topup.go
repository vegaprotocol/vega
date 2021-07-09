package execution

import (
	"context"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/types"
	"code.vegaprotocol.io/data-node/types/num"
)

func (m *Market) checkBondBalance(ctx context.Context) {
	lps := m.liquidity.ProvisionsPerParty().Slice()
	mID := m.GetID()
	asset, _ := m.mkt.GetAsset()
	transfers := make([]*types.TransferResponse, 0, len(lps))
	for _, lp := range lps {
		party := lp.PartyId
		bondAcc, err := m.collateral.GetPartyBondAccount(mID, party, asset)
		if err != nil || bondAcc == nil {
			continue
		}
		// commitment is covered by bond balance already
		if bondAcc.Balance.GTE(lp.CommitmentAmount) {
			continue
		}
		gen, err := m.collateral.GetPartyGeneralAccount(party, asset)
		// no balance in general account
		if err != nil || gen.Balance.IsZero() {
			continue
		}
		bondShort := num.Zero().Sub(lp.CommitmentAmount, bondAcc.Balance)
		// Min clones
		amt := num.Min(bondShort, gen.Balance)
		t := &types.Transfer{
			Owner: party,
			Type:  types.TransferType_TRANSFER_TYPE_BOND_LOW,
			Amount: &types.FinancialAmount{
				Asset:  asset,
				Amount: amt,
			},
			MinAmount: amt.Clone(),
		}
		resp, err := m.collateral.BondUpdate(ctx, mID, t)
		if err != nil {
			m.log.Panic("Failed to top up bond balance",
				logging.String("market-id", mID),
				logging.String("party", party),
				logging.Error(err))
		}
		if len(resp.Transfers) > 0 {
			transfers = append(transfers, resp)
		}
	}
	if len(transfers) > 0 {
		m.broker.Send(events.NewTransferResponse(ctx, transfers))
	}
}
