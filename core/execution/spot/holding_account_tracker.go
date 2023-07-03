package spot

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"golang.org/x/exp/maps"
)

type HoldingAccountTracker struct {
	orderIDToQuantity map[string]*num.Uint
	orderIDToFee      map[string]*num.Uint
	collateral        common.Collateral
	stopped           bool
	log               *logging.Logger
	snapshot          *types.PayloadHoldingAccountTracker
}

func NewHoldingAccountTracker(marketID string, log *logging.Logger, collateral common.Collateral) *HoldingAccountTracker {
	return &HoldingAccountTracker{
		orderIDToQuantity: map[string]*num.Uint{},
		orderIDToFee:      map[string]*num.Uint{},
		collateral:        collateral,
		log:               log,
		snapshot: &types.PayloadHoldingAccountTracker{
			HoldingAccountTracker: &types.HoldingAccountTracker{
				MarketID: marketID,
			},
		},
	}
}

func (hat *HoldingAccountTracker) GetCurrentHolding(orderID string) (*num.Uint, *num.Uint) {
	qty := num.UintZero()
	fees := num.UintZero()
	if q, ok := hat.orderIDToQuantity[orderID]; ok {
		qty = q
	}
	if f, ok := hat.orderIDToFee[orderID]; ok {
		fees = f
	}
	return qty, fees
}

func (hat *HoldingAccountTracker) TransferToHoldingAccount(ctx context.Context, orderID, party, asset string, quantity *num.Uint, fee *num.Uint) (*types.LedgerMovement, error) {
	if _, ok := hat.orderIDToQuantity[orderID]; ok {
		return nil, fmt.Errorf("funds for the order have already been transferred to the holding account")
	}
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
	if fee != nil && !fee.IsZero() {
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
	hat.orderIDToQuantity[orderID] = num.UintZero().Sub(lockedQuantity, quantity)
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

func (hat *HoldingAccountTracker) StopSnapshots() {
	hat.stopped = true
}

func (hat *HoldingAccountTracker) Keys() []string {
	return []string{hat.snapshot.Key()}
}

func (hat *HoldingAccountTracker) Stopped() bool {
	return hat.stopped
}

func (hat *HoldingAccountTracker) Namespace() types.SnapshotNamespace {
	return types.HoldingAccountTrackerSnapshot
}

func (hat *HoldingAccountTracker) GetState(key string) ([]byte, []types.StateProvider, error) {
	if key != hat.snapshot.Key() {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if hat.stopped {
		return nil, nil, nil
	}
	payload := hat.buildPayload()

	s, err := proto.Marshal(payload.IntoProto())
	return s, nil, err
}

func (hat *HoldingAccountTracker) buildPayload() *types.Payload {
	quantities := make([]*types.HoldingAccountQuantity, 0, len(hat.orderIDToQuantity))

	orderIDs := map[string]struct{}{}
	for k := range hat.orderIDToQuantity {
		orderIDs[k] = struct{}{}
	}
	for k := range hat.orderIDToFee {
		orderIDs[k] = struct{}{}
	}
	orderIDSlice := maps.Keys(orderIDs)
	sort.Strings(orderIDSlice)

	for _, oid := range orderIDSlice {
		quantities = append(quantities, &types.HoldingAccountQuantity{
			ID:          oid,
			Quantity:    hat.orderIDToQuantity[oid],
			FeeQuantity: hat.orderIDToFee[oid],
		})
	}

	return &types.Payload{
		Data: &types.PayloadHoldingAccountTracker{
			HoldingAccountTracker: &types.HoldingAccountTracker{
				MarketID:                 hat.snapshot.HoldingAccountTracker.MarketID,
				HoldingAccountQuantities: quantities,
			},
		},
	}
}

func (hat *HoldingAccountTracker) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if hat.Namespace() != payload.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	var at *types.HoldingAccountTracker

	switch pl := payload.Data.(type) {
	case *types.PayloadHoldingAccountTracker:
		at = pl.HoldingAccountTracker
	default:
		return nil, types.ErrUnknownSnapshotType
	}

	for _, haq := range at.HoldingAccountQuantities {
		if haq.FeeQuantity != nil {
			hat.orderIDToFee[haq.ID] = haq.FeeQuantity
		}
		if haq.Quantity != nil {
			hat.orderIDToQuantity[haq.ID] = haq.Quantity
		}
	}

	return nil, nil
}
