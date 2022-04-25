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

// marketFeesTracker tracks the amount of fees paid/received by different parties in the market.
type marketFeesTracker struct {
	makerFees map[string]*num.Uint
	takerFees map[string]*num.Uint
	lpFees    map[string]*num.Uint
}

func (mft *marketFeesTracker) addReceivedMakerFees(maker string, amount *num.Uint) {
	mft.addFees(mft.makerFees, maker, amount)
}

func (mft *marketFeesTracker) addPaidTakerFees(taker string, amount *num.Uint) {
	mft.addFees(mft.takerFees, taker, amount)
}

func (mft *marketFeesTracker) addReceivedLPFees(lp string, amount *num.Uint) {
	mft.addFees(mft.lpFees, lp, amount)
}

func (mft *marketFeesTracker) addFees(m map[string]*num.Uint, party string, amount *num.Uint) {
	if _, ok := m[party]; !ok {
		m[party] = amount.Clone()
		return
	}
	m[party] = num.Sum(m[party], amount)
}

// FeesTracker tracks how much fees are paid and received for a market by parties by epoch.
type FeesTracker struct {
	marketToTracker map[string]*marketFeesTracker
	currentEpoch    uint64
	ss              *snapshotState
}

// NewFeesTracker instantiates the fees tracker.
func NewFeesTracker(epochEngine EpochEngine) *FeesTracker {
	ft := &FeesTracker{
		marketToTracker: map[string]*marketFeesTracker{},
		ss:              &snapshotState{changed: true},
	}
	epochEngine.NotifyOnEpoch(ft.onEpochEvent, ft.onEpochRestore)
	return ft
}

// onEpochEvent is called when the state of the epoch changes, we only care about new epochs starting.
func (f *FeesTracker) onEpochEvent(_ context.Context, epoch types.Epoch) {
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_START {
		f.marketToTracker = map[string]*marketFeesTracker{}
		f.ss.changed = true
	}
	f.currentEpoch = epoch.Seq
}

// GetFeePartyScores returns the fraction each of the participants paid/received in the given fee of the market in the relevant period.
func (f *FeesTracker) GetFeePartyScores(market string, feeType types.TransferType) []*types.FeePartyScore {
	if _, ok := f.marketToTracker[market]; !ok {
		return []*types.FeePartyScore{}
	}

	feesData := map[string]*num.Uint{}

	switch feeType {
	case types.TransferTypeMakerFeeReceive:
		feesData = f.marketToTracker[market].makerFees
	case types.TransferTypeMakerFeePay:
		feesData = f.marketToTracker[market].takerFees
	case types.TransferTypeLiquidityFeeDistribute:
		feesData = f.marketToTracker[market].lpFees
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

// ensureMarketFeesTracker returns the market tracker for the given market if it exists or creates a new one and saves it if it doesn't.
func (f *FeesTracker) ensureMarketFeesTracker(market string) *marketFeesTracker {
	if mft, ok := f.marketToTracker[market]; ok {
		return mft
	}
	mft := &marketFeesTracker{
		makerFees: map[string]*num.Uint{},
		takerFees: map[string]*num.Uint{},
		lpFees:    map[string]*num.Uint{},
	}
	f.marketToTracker[market] = mft
	f.ss.changed = true
	return mft
}

// UpdateFeesFromTransfers takes a slice of transfers and if they represent fees it updates the market fee tracker.
func (f *FeesTracker) UpdateFeesFromTransfers(market string, transfers []*types.Transfer) {
	for _, t := range transfers {
		switch t.Type {
		case types.TransferTypeMakerFeePay:
			f.ensureMarketFeesTracker(market).addPaidTakerFees(t.Owner, t.Amount.Amount)
		case types.TransferTypeMakerFeeReceive:
			f.ensureMarketFeesTracker(market).addReceivedMakerFees(t.Owner, t.Amount.Amount)
		case types.TransferTypeLiquidityFeeDistribute:
			f.ensureMarketFeesTracker(market).addReceivedLPFees(t.Owner, t.Amount.Amount)
		default:
		}
	}
}
