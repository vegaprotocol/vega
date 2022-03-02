package execution

import (
	"context"
	"sort"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/epoch_service_mock.go -package mocks code.vegaprotocol.io/vega/execution EpochEngine
type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
}

// assetFeesTracker tracks the amount of fees paid/received by different parties in the asset.
type assetFeesTracker struct {
	makerFees map[string]*num.Uint
	takerFees map[string]*num.Uint
	lpFees    map[string]*num.Uint
}

func (aft *assetFeesTracker) addReceivedMakerFees(maker string, amount *num.Uint) {
	aft.addFees(aft.makerFees, maker, amount)
}

func (aft *assetFeesTracker) addPaidTakerFees(taker string, amount *num.Uint) {
	aft.addFees(aft.takerFees, taker, amount)
}

func (aft *assetFeesTracker) addReceivedLPFees(lp string, amount *num.Uint) {
	aft.addFees(aft.lpFees, lp, amount)
}

func (aft *assetFeesTracker) addFees(m map[string]*num.Uint, party string, amount *num.Uint) {
	if _, ok := m[party]; !ok {
		m[party] = amount.Clone()
		return
	}
	m[party] = num.Sum(m[party], amount)
}

// FeesTracker tracks how much fees are paid and received for assets by parties over epochs.
type FeesTracker struct {
	assetToTracker map[string]*assetFeesTracker
	currentEpoch   uint64
	ss             *snapshotState
}

// NewFeesTracker instantiates the fees tracker.
func NewFeesTracker(epochEngine EpochEngine) *FeesTracker {
	ft := &FeesTracker{
		assetToTracker: map[string]*assetFeesTracker{},
		ss:             &snapshotState{changed: true},
	}
	epochEngine.NotifyOnEpoch(ft.onEpochEvent, ft.onEpochRestore)
	return ft
}

// onEpochEvent is called when the state of the epoch changes, we only care about new epochs starting.
func (f *FeesTracker) onEpochEvent(_ context.Context, epoch types.Epoch) {
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_START {
		f.assetToTracker = map[string]*assetFeesTracker{}
	}
	f.currentEpoch = epoch.Seq
}

// GetFeePartyScores returns the fraction each of the participants paid/received in the given fee of the asset in the relevant period.
func (f *FeesTracker) GetFeePartyScores(asset string, feeType types.TransferType) []*types.FeePartyScore {
	if _, ok := f.assetToTracker[asset]; !ok {
		return []*types.FeePartyScore{}
	}

	feesData := map[string]*num.Uint{}

	switch feeType {
	case types.TransferTypeMakerFeeReceive:
		feesData = f.assetToTracker[asset].makerFees
	case types.TransferTypeMakerFeePay:
		feesData = f.assetToTracker[asset].takerFees
	case types.TransferTypeLiquidityFeeDistribute:
		feesData = f.assetToTracker[asset].lpFees
	default:
	}

	scores := make([]*types.FeePartyScore, 0, len(feesData))
	parties := make([]string, 0, len(scores))
	for party := range feesData {
		parties = append(parties, party)
	}
	sort.Strings(parties)

	total := num.DecimalZero()
	for _, party := range parties {
		total = total.Add(feesData[party].ToDecimal())
	}
	for _, party := range parties {
		scores = append(scores, &types.FeePartyScore{Party: party, Score: feesData[party].ToDecimal().Div(total)})
	}
	return scores
}

// ensureAssetFeesTracker returns the asset tracker for the given asset if it exists or creates a new one and saves it if it doesn't.
func (f *FeesTracker) ensureAssetFeesTracker(asset string) *assetFeesTracker {
	if aft, ok := f.assetToTracker[asset]; ok {
		return aft
	}
	aft := &assetFeesTracker{
		makerFees: map[string]*num.Uint{},
		takerFees: map[string]*num.Uint{},
		lpFees:    map[string]*num.Uint{},
	}
	f.assetToTracker[asset] = aft
	return aft
}

// UpdateFeesFromTransfers takes a slice of transfers and if they represent fees it updates the asset fee tracker.
func (f *FeesTracker) UpdateFeesFromTransfers(transfers []*types.Transfer) {
	for _, t := range transfers {
		asset := t.Amount.Asset

		switch t.Type {
		case types.TransferTypeMakerFeePay:
			f.ensureAssetFeesTracker(asset).addPaidTakerFees(t.Owner, t.Amount.Amount)
		case types.TransferTypeMakerFeeReceive:
			f.ensureAssetFeesTracker(asset).addReceivedMakerFees(t.Owner, t.Amount.Amount)
		case types.TransferTypeLiquidityFeeDistribute:
			f.ensureAssetFeesTracker(asset).addReceivedLPFees(t.Owner, t.Amount.Amount)
		default:
		}
	}
}
