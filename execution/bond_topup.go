package execution

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
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
		if bondAcc.Balance >= lp.CommitmentAmount {
			continue
		}
		gen, err := m.collateral.GetPartyGeneralAccount(party, asset)
		// no balance in general account
		if err != nil || gen.Balance == 0 {
			continue
		}
		bondShort := lp.CommitmentAmount - bondAcc.Balance
		amt := min(bondShort, gen.Balance)
		t := &types.Transfer{
			Owner: party,
			Type:  types.TransferType_TRANSFER_TYPE_BOND_LOW,
			Amount: &types.FinancialAmount{
				Asset:  asset,
				Amount: amt,
			},
			MinAmount: amt,
		}
		fmt.Printf("top-up for party %s: short %d, transfer for %d\n", party, bondShort, amt)
		resp, err := m.collateral.BondUpdate(ctx, mID, party, t)
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

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
