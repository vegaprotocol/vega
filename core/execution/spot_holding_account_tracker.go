package execution

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type HoldingAccountTracker struct {
	orderIDToQuantity map[string]*num.Uint
	orderIDToFee      map[string]*num.Uint
	collateral        SpotMarketCollateral
}

func NewHoldingAccountTracker(collateral SpotMarketCollateral) *HoldingAccountTracker {
	return &HoldingAccountTracker{
		orderIDToQuantity: map[string]*num.Uint{},
		orderIDToFee:      map[string]*num.Uint{},
		collateral:        collateral,
	}
}

func (hat *HoldingAccountTracker) getCurrentHolding(orderID string) (*num.Uint, *num.Uint) {
	fees := num.UintZero()
	qty := num.UintZero()
	if f, ok := hat.orderIDToFee[orderID]; ok {
		fees = f
	}
	if q, ok := hat.orderIDToQuantity[orderID]; ok {
		qty = q
	}
	return qty, fees
}

func (hat *HoldingAccountTracker) TransferToHoldingAccount(ctx context.Context, orderID, party, asset string, quantity *num.Uint, fee *num.Uint) (*types.LedgerMovement, error) {
	total := num.Sum(quantity, fee)

	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Asset:  asset,
			Amount: total,
		},
		Type: types.TransferTypeHoldingAccount,
	}
	le, err := hat.collateral.TransferToHoldingAccount(ctx, transfer)
	if err != nil {
		return nil, err
	}
	hat.orderIDToQuantity[orderID] = quantity
	if !fee.IsZero() {
		hat.orderIDToFee[orderID] = fee
	}
	return le, nil
}

func (hat *HoldingAccountTracker) TransferFeeToHoldingAccount(ctx context.Context, orderID, party, asset string, feeQuantity *num.Uint) (*types.LedgerMovement, error) {
	if feeQuantity.IsZero() {
		return nil, nil
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Asset:  asset,
			Amount: feeQuantity.Clone(),
		},
		Type: types.TransferTypeHoldingAccount,
	}
	le, err := hat.collateral.TransferToHoldingAccount(ctx, transfer)
	if err != nil {
		return nil, err
	}
	hat.orderIDToFee[orderID] = feeQuantity
	return le, nil
}

func (hat *HoldingAccountTracker) ReleaseFeeFromHoldingAccount(ctx context.Context, orderID, party, asset string) (*types.LedgerMovement, error) {
	feeQuantity, ok := hat.orderIDToFee[orderID]
	if !ok {
		return nil, fmt.Errorf("failed to find locked fee amount for order id %s", orderID)
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Asset:  asset,
			Amount: feeQuantity.Clone(),
		},
		Type: types.TransferTypeHoldingAccount,
	}
	delete(hat.orderIDToFee, orderID)
	le, err := hat.collateral.ReleaseFromHoldingAccount(ctx, transfer)
	if err != nil {
		return nil, err
	}
	return le, err
}

func (hat *HoldingAccountTracker) ReleaseQuantityHoldingAccount(ctx context.Context, orderID, party, asset string, quantity *num.Uint, fee *num.Uint) (*types.LedgerMovement, error) {
	total := num.Sum(quantity, fee)
	if !fee.IsZero() {
		lockedFee, ok := hat.orderIDToFee[orderID]
		if !ok || lockedFee.LT(fee) {
			return nil, fmt.Errorf("insufficient locked fee to release for order %s", orderID)
		}
		hat.orderIDToFee[orderID] = num.UintZero().Sub(lockedFee, fee)
	}
	lockedQuantity, ok := hat.orderIDToQuantity[orderID]
	if !ok || lockedQuantity.LT(quantity) {
		return nil, fmt.Errorf("insufficient locked quantity to release for order %s", orderID)
	}
	hat.orderIDToFee[orderID] = num.UintZero().Sub(lockedQuantity, quantity)
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Asset:  asset,
			Amount: total,
		},
		Type: types.TransferTypeReleaseHoldingAccount,
	}
	le, err := hat.collateral.ReleaseFromHoldingAccount(ctx, transfer)
	if err != nil {
		return nil, err
	}
	return le, err
}

func (hat *HoldingAccountTracker) ReleaseAllFromHoldingAccount(ctx context.Context, orderID, party, asset string) (*types.LedgerMovement, error) {
	fee := num.UintZero()
	amt := num.UintZero()
	if f, ok := hat.orderIDToFee[orderID]; ok {
		fee = f
	}
	if a, ok := hat.orderIDToQuantity[orderID]; ok {
		amt = a
	}

	total := num.Sum(fee, amt)
	delete(hat.orderIDToFee, orderID)
	delete(hat.orderIDToQuantity, orderID)

	if total.IsZero() {
		return nil, nil
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Asset:  asset,
			Amount: total,
		},
		Type: types.TransferTypeReleaseHoldingAccount,
	}
	return hat.collateral.ReleaseFromHoldingAccount(ctx, transfer)
}
