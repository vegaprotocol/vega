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

package future

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

// ErrBondSlashing - just indicates that we had to penalize the party due to insufficient funds, and as such, we have to cancel their LP.
var ErrBondSlashing = errors.New("bond slashing")

func (m *Market) transferMargins(ctx context.Context, risk []events.Risk, closed []events.MarketPosition) error {
	if m.as.InAuction() {
		return m.transferMarginsAuction(ctx, risk, closed)
	}
	return m.transferMarginsContinuous(ctx, risk)
}

func (m *Market) transferMarginsAuction(ctx context.Context, risk []events.Risk, distressed []events.MarketPosition) error {
	evts := make([]events.Event, 0, len(risk))
	mID := m.GetID()
	// first, update the margin accounts for all parties who have enough balance
	for _, re := range risk {
		tr, _, err := m.collateral.MarginUpdateOnOrder(ctx, mID, re)
		if err != nil {
			// @TODO handle this
			return err
		}
		if len(tr.Entries) > 0 {
			evts = append(evts, events.NewLedgerMovements(ctx, []*types.LedgerMovement{tr}))
		}
	}

	m.broker.SendBatch(evts)
	rmorders, err := m.matching.RemoveDistressedOrders(distressed)
	if err != nil {
		return err
	}
	evts = make([]events.Event, 0, len(rmorders))
	for _, o := range rmorders {
		// cancel order
		o.Status = types.OrderStatusCancelled
		o.Reason = types.OrderErrorInsufficientAssetBalance
		// create event
		evts = append(evts, events.NewOrderEvent(ctx, o))
		// remove order from positions
		_ = m.position.UnregisterOrder(ctx, o)
	}
	m.broker.SendBatch(evts)
	return nil
}

func (m *Market) transferRecheckMargins(ctx context.Context, risk []events.Risk) {
	if len(risk) == 0 {
		return
	}
	mID := m.GetID()
	evts := make([]events.Event, 0, len(risk))
	for _, r := range risk {
		var tr *types.LedgerMovement
		responses := make([]*types.LedgerMovement, 0, 1)
		tr, closed, err := m.collateral.MarginUpdateOnOrder(ctx, mID, r)
		if err != nil {
			m.log.Warn("margin recheck failed",
				logging.MarketID(m.GetID()),
				logging.PartyID(r.Party()),
				logging.Error(err))
		}
		if tr != nil {
			responses = append(responses, tr)
		}
		if closed != nil && !closed.MarginShortFall().IsZero() {
			resp, err := m.bondSlashing(ctx, closed)
			if err != nil {
				m.log.Panic("Bond slashing for non-distressed LP failed",
					logging.String("party", closed.Party()),
					logging.Error(err),
				)
			}
			responses = append(responses, resp...)
		}
		if len(responses) > 0 {
			evts = append(evts, events.NewLedgerMovements(ctx, responses))
		}
	}
	m.broker.SendBatch(evts)
}

func (m *Market) transferMarginsContinuous(ctx context.Context, risk []events.Risk) error {
	if len(risk) > 1 {
		return errors.New("transferMarginsContinuous should not be possible when len(risk) > 1")
	}
	if len(risk) == 0 {
		return nil
	}
	mID := m.GetID()
	tr, closed, err := m.collateral.MarginUpdateOnOrder(ctx, mID, risk[0])
	if err != nil {
		return err
	}
	// if LP shortfall is not empty, this party will have to pay the LP penalty
	responses := make([]*types.LedgerMovement, 0, len(risk))
	if tr != nil {
		responses = append(responses, tr)
	}
	// margin shortfall && liquidity provider -> bond slashing
	if closed != nil && !closed.MarginShortFall().IsZero() {
		// get bond penalty
		resp, err := m.bondSlashing(ctx, closed)
		if err != nil {
			return err
		}
		responses = append(responses, resp...)
	}
	if len(responses) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, responses))
	}

	return nil
}

func (m *Market) bondSlashing(ctx context.Context, closed ...events.Margin) ([]*types.LedgerMovement, error) {
	mID := m.GetID()
	ret := make([]*types.LedgerMovement, 0, len(closed))
	for _, c := range closed {
		if !m.liquidityEngine.IsLiquidityProvider(c.Party()) {
			continue
		}
		penalty, _ := num.UintFromDecimal(
			num.DecimalFromUint(c.MarginShortFall()).Mul(m.bondPenaltyFactor).Floor(),
		)

		resp, err := m.collateral.BondUpdate(ctx, mID, &types.Transfer{
			Owner: c.Party(),
			Amount: &types.FinancialAmount{
				Amount: penalty,
				Asset:  c.Asset(),
			},
			Type:      types.TransferTypeBondSlashing,
			MinAmount: num.UintZero(),
		})
		if err != nil {
			return nil, err
		}
		ret = append(ret, resp)
	}
	return ret, nil
}
