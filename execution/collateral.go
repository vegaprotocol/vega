package execution

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrBondSlashing - just indicates that we had to penalize the trader due to insufficient funds, and as such, we have to cancel their LP
	ErrBondSlashing = errors.New("bond slashing")
)

func (m *Market) transferMargins(ctx context.Context, risk []events.Risk, closed []events.MarketPosition) error {
	if m.as.InAuction() {
		return m.transferMarginsAuction(ctx, risk, closed)
	}
	return m.transferMarginsContinuous(ctx, risk, closed)
}

func (m *Market) transferMarginsAuction(ctx context.Context, risk []events.Risk, distressed []events.MarketPosition) error {
	evts := make([]events.Event, 0, len(risk))
	// asset, _ := m.mkt.GetAsset()
	mID := m.GetID()
	// first, update the margin accounts for all traders who have enough balance
	for _, re := range risk {
		tr, _, err := m.collateral.MarginUpdateOnOrder(ctx, mID, re)
		if err != nil {
			// @TODO handle this
			return err
		}
		evts = append(evts, events.NewTransferResponse(ctx, []*types.TransferResponse{tr}))
	}
	m.broker.SendBatch(evts)
	rmorders, err := m.matching.RemoveDistressedOrders(distressed)
	if err != nil {
		return err
	}
	evts = make([]events.Event, 0, len(rmorders))
	for _, o := range rmorders {
		// cancel order
		o.Status = types.Order_STATUS_CANCELLED
		o.Reason = types.OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE
		// create event
		evts = append(evts, events.NewOrderEvent(ctx, o))
		// remove order from positions
		m.position.UnregisterOrder(o)
	}
	m.broker.SendBatch(evts)
	return nil
}

func (m *Market) transferMarginsContinuous(ctx context.Context, risk []events.Risk, lpShortfall []events.MarketPosition) error {
	if len(risk) > 1 {
		return errors.New("this should not be possiburu")
	}
	if len(risk) == 0 {
		return nil
	}
	mID := m.GetID()
	asset, _ := m.mkt.GetAsset()
	tr, closed, err := m.collateral.MarginUpdateOnOrder(ctx, mID, risk[0])
	if err != nil {
		return err
	}
	rEvt := risk[0]
	// if LP shortfall is not empty, this trader will have to pay the LP penalty
	responses := make([]*types.TransferResponse, 0, len(risk)+len(lpShortfall))
	if tr != nil {
		responses = append(responses, tr)
	}
	var rerr error
	// margin shortfall && liquidity provider -> bond slashing
	if closed != nil && len(lpShortfall) != 0 {
		// get bond penalty
		rerr = ErrBondSlashing
		penalty := m.bondPenaltyFactor * float64(closed.MarginShortFall())
		tr := types.Transfer{
			Owner: rEvt.Party(),
			Amount: &types.FinancialAmount{
				Amount: uint64(penalty),
				Asset:  asset,
			},
			Type:      types.TransferType_TRANSFER_TYPE_BOND_SLASHING,
			MinAmount: uint64(penalty),
		}
		resp, err := m.collateral.BondUpdate(ctx, mID, rEvt.Party(), &tr)
		if err != nil {
			return err
		}
		responses = append(responses, resp)
	}
	m.broker.Send(events.NewTransferResponse(ctx, responses))
	return rerr
}
