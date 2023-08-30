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

package common

import (
	"context"
	"errors"
	"sort"
	"time"

	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	key                        = (&types.PayloadMarketActivityTracker{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for market activity tracker snapshot")
	hashKeys                   = []string{key}
)

type snapshotState struct {
	serialised []byte
}

func (mat *MarketActivityTracker) Namespace() types.SnapshotNamespace {
	return types.MarketActivityTrackerSnapshot
}

func (mat *MarketActivityTracker) Keys() []string {
	return hashKeys
}

func (mat *MarketActivityTracker) Stopped() bool {
	return false
}

func totalToProto(data *feeData) *checkpoint.DataWithHistory {
	d := &checkpoint.DataWithHistory{
		RunningTotal:        data.runningTotal.String(),
		PreviousEpochs:      make([]string, 0, len(data.previousEpochs)),
		PreviousEpochsIndex: uint64(data.previousEpochsIdx),
	}
	for _, u := range data.previousEpochs {
		v := ""
		if u != nil {
			v = u.String()
		}
		d.PreviousEpochs = append(d.PreviousEpochs, v)
	}
	return d
}

func returnsDataToProto(partyM2MData map[string]*m2mData) []*checkpoint.ReturnsData {
	parties := make([]string, 0, len(partyM2MData))
	for k := range partyM2MData {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	data := make([]*checkpoint.ReturnsData, 0, len(parties))
	for _, party := range parties {
		pd := partyM2MData[party]
		pdProto := &checkpoint.ReturnsData{
			Party: party,
			Data: &checkpoint.DataWithHistory{
				RunningTotal:        pd.runningTotal.String(),
				PreviousEpochs:      make([]string, 0, len(pd.previousEpochs)),
				PreviousEpochsIndex: uint64(pd.previousEpochsIdx),
			},
		}
		for _, u := range pd.previousEpochs {
			pdProto.Data.PreviousEpochs = append(pdProto.Data.PreviousEpochs, u.String())
		}
		data = append(data, pdProto)
	}
	return data
}

func twNotionalToProto(twNotional map[string]*twNotionalPosition) []*checkpoint.TWNotionalPosition {
	parties := make([]string, 0, len(twNotional))
	for k := range twNotional {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	data := make([]*checkpoint.TWNotionalPosition, 0, len(parties))
	for _, party := range parties {
		pd := twNotional[party]
		pdProto := &checkpoint.TWNotionalPosition{
			Party:              party,
			Position:           pd.position.String(),
			Price:              pd.price.String(),
			Time:               pd.t.UnixNano(),
			TwNotionalPosition: pd.currentEpochTWNotional.String(),
		}
		data = append(data, pdProto)
	}
	return data
}

func positionHistoryToProto(partyPositionHistory map[string]*twPosition) []*checkpoint.TWPositionData {
	parties := make([]string, 0, len(partyPositionHistory))
	for k := range partyPositionHistory {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	data := make([]*checkpoint.TWPositionData, 0, len(parties))
	for _, party := range parties {
		pd := partyPositionHistory[party]
		pdProto := &checkpoint.TWPositionData{
			Party:    party,
			Position: pd.position.String(),
			Time:     pd.t.UnixNano(),
			Data: &checkpoint.DataWithHistory{
				RunningTotal:        pd.currentEpochTWPosition.String(),
				PreviousEpochs:      make([]string, 0, len(pd.previousEpochs)),
				PreviousEpochsIndex: uint64(pd.previousEpochsIdx),
			},
		}
		for _, u := range pd.previousEpochs {
			pdProto.Data.PreviousEpochs = append(pdProto.Data.PreviousEpochs, u.String())
		}
		data = append(data, pdProto)
	}
	return data
}

func takerNotionalToProto(takerNotional map[string]*num.Uint) []*checkpoint.TakerNotionalVolume {
	ret := make([]*checkpoint.TakerNotionalVolume, 0, len(takerNotional))
	for k, u := range takerNotional {
		ret = append(ret, &checkpoint.TakerNotionalVolume{Party: k, Volume: u.String()})
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Party < ret[j].Party
	})
	return ret
}

func marketFeesToProto(partyFees map[string]*feeData) []*checkpoint.PartyFeeData {
	parties := make([]string, 0, len(partyFees))
	for k := range partyFees {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	pf := make([]*checkpoint.PartyFeeData, 0, len(parties))
	for _, party := range parties {
		pfd := partyFees[party]
		pfdProto := &checkpoint.PartyFeeData{
			Party: party,
			Data: &checkpoint.DataWithHistory{
				RunningTotal:        pfd.runningTotal.String(),
				PreviousEpochs:      make([]string, 0, len(pfd.previousEpochs)),
				PreviousEpochsIndex: uint64(pfd.previousEpochsIdx),
			},
		}
		for _, u := range pfd.previousEpochs {
			v := ""
			if u != nil {
				v = u.String()
			}
			pfdProto.Data.PreviousEpochs = append(pfdProto.Data.PreviousEpochs, v)
		}
		pf = append(pf, pfdProto)
	}
	return pf
}

func (mt *marketTracker) IntoProto(market string) *checkpoint.MarketActivityTracker {
	paid := make([]string, 0, len(mt.proposersPaid))
	for k := range mt.proposersPaid {
		paid = append(paid, k)
	}
	sort.Strings(paid)

	return &checkpoint.MarketActivityTracker{
		Asset:                        mt.asset,
		Market:                       market,
		MakerFeesReceivedHistory:     marketFeesToProto(mt.makerFeesReceived),
		MakerFeesPaidHistory:         marketFeesToProto(mt.makerFeesReceived),
		LpFeesHistory:                marketFeesToProto(mt.lpFees),
		TimeWeightedPositionHistory:  positionHistoryToProto(mt.timeWeightedPosition),
		TimeWeightedNotionalPosition: twNotionalToProto(mt.twNotionalPosition),
		ReturnsData:                  returnsDataToProto(mt.partyM2M),
		TotalMakerFeesReceived:       totalToProto(mt.totalMakerFeesReceived),
		TotalMakerFeesPaid:           totalToProto(mt.totalMakerFeesPaid),
		TotalLpFees:                  totalToProto(mt.totalLpFees),
		ValueTraded:                  mt.valueTraded.String(),
		Proposer:                     mt.proposer,
		BonusPaid:                    paid,
		ReadyToDelete:                mt.readyToDelete,
	}
}

func (mat *MarketActivityTracker) serialiseFeesTracker() *snapshot.MarketTracker {
	marketActivity := []*checkpoint.MarketActivityTracker{}
	assets := make([]string, 0, len(mat.assetToMarketTrackers))
	for k := range mat.assetToMarketTrackers {
		assets = append(assets, k)
	}
	sort.Strings(assets)
	for _, asset := range assets {
		markets := make([]string, 0, len(mat.assetToMarketTrackers[asset]))
		assetMarketTrackers := mat.assetToMarketTrackers[asset]
		for k := range mat.assetToMarketTrackers[asset] {
			markets = append(markets, k)
		}
		sort.Strings(markets)

		for _, market := range markets {
			marketActivity = append(marketActivity, assetMarketTrackers[market].IntoProto(market))
		}
	}

	return &snapshot.MarketTracker{
		MarketActivity:      marketActivity,
		TakerNotionalVolume: takerNotionalToProto(mat.partyTakerNotionalVolume),
	}
}

// get the serialised form and hash of the given key.
func (mat *MarketActivityTracker) serialise(k string) ([]byte, error) {
	if k != key {
		return nil, ErrSnapshotKeyDoesNotExist
	}
	payload := types.Payload{
		Data: &types.PayloadMarketActivityTracker{
			MarketActivityData: mat.serialiseFeesTracker(),
		},
	}
	x := payload.IntoProto()
	data, err := proto.Marshal(x)
	if err != nil {
		return nil, err
	}

	mat.ss.serialised = data
	return data, nil
}

func (mat *MarketActivityTracker) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := mat.serialise(k)
	return state, nil, err
}

func (mat *MarketActivityTracker) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if mat.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadMarketActivityTracker:
		mat.restore(pl.MarketActivityData)
		var err error
		mat.ss.serialised, err = proto.Marshal(p.IntoProto())
		return nil, err
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func marketTrackerFromProto(tracker *checkpoint.MarketActivityTracker) *marketTracker {
	valueTrades, _ := num.UintFromString(tracker.ValueTraded, 10)
	mft := &marketTracker{
		makerFeesReceived:      map[string]*feeData{},
		makerFeesPaid:          map[string]*feeData{},
		lpFees:                 map[string]*feeData{},
		timeWeightedPosition:   map[string]*twPosition{},
		partyM2M:               map[string]*m2mData{},
		twNotionalPosition:     map[string]*twNotionalPosition{},
		totalMakerFeesReceived: &feeData{},
		totalMakerFeesPaid:     &feeData{},
		totalLpFees:            &feeData{},
		valueTraded:            valueTrades,
		proposer:               tracker.Proposer,
		proposersPaid:          map[string]struct{}{},
		asset:                  tracker.Asset,
		readyToDelete:          tracker.ReadyToDelete,
	}

	for _, bpfpa := range tracker.BonusPaid {
		mft.proposersPaid[bpfpa] = struct{}{}
	}

	// legacy support for upgrading from an old snapshot/checkpoint
	if len(tracker.MakerFeesReceived) > 0 {
		total := num.UintZero()
		for _, mf := range tracker.MakerFeesReceived {
			fd := &feeData{
				previousEpochs:    make([]*num.Uint, 0, maxWindowSize),
				previousEpochsIdx: 0,
			}
			fd.runningTotal, _ = num.UintFromString(mf.Fee, 10)
			total.AddSum(fd.runningTotal)
			mft.makerFeesReceived[mf.Party] = fd
		}
		mft.totalMakerFeesReceived = &feeData{
			runningTotal:      total,
			previousEpochs:    make([]*num.Uint, 0, maxWindowSize),
			previousEpochsIdx: 0,
		}
	}

	if len(tracker.MakerFeesPaid) > 0 {
		total := num.UintZero()
		for _, mf := range tracker.MakerFeesPaid {
			fd := &feeData{
				previousEpochs:    make([]*num.Uint, 0, maxWindowSize),
				previousEpochsIdx: 0,
			}
			fd.runningTotal, _ = num.UintFromString(mf.Fee, 10)
			total.AddSum(fd.runningTotal)
			mft.makerFeesPaid[mf.Party] = fd
		}
		mft.totalMakerFeesPaid = &feeData{
			runningTotal:      total,
			previousEpochs:    make([]*num.Uint, 0, maxWindowSize),
			previousEpochsIdx: 0,
		}
	}

	if len(tracker.LpFees) > 0 {
		total := num.UintZero()
		for _, mf := range tracker.LpFees {
			fd := &feeData{
				previousEpochs:    make([]*num.Uint, 0, maxWindowSize),
				previousEpochsIdx: 0,
			}
			fd.runningTotal, _ = num.UintFromString(mf.Fee, 10)
			total.AddSum(fd.runningTotal)
			mft.makerFeesPaid[mf.Party] = fd
		}
		mft.totalLpFees = &feeData{
			runningTotal:      total,
			previousEpochs:    make([]*num.Uint, 0, maxWindowSize),
			previousEpochsIdx: 0,
		}
	}
	// end of legacy support

	if len(tracker.TimeWeightedNotionalPosition) > 0 {
		for _, tp := range tracker.TimeWeightedNotionalPosition {
			position, _ := num.DecimalFromString(tp.Position)
			price, _ := num.UintFromString(tp.Price, 10)
			currentEpochTWNotional, _ := num.DecimalFromString(tp.TwNotionalPosition)
			data := &twNotionalPosition{
				t:                      time.Unix(0, tp.Time),
				position:               position,
				price:                  price,
				currentEpochTWNotional: currentEpochTWNotional,
			}
			mft.twNotionalPosition[tp.Party] = data
		}
	}

	if len(tracker.TimeWeightedPositionHistory) > 0 {
		for _, td := range tracker.TimeWeightedPositionHistory {
			position, _ := num.DecimalFromString(td.Position)
			current, _ := num.DecimalFromString(td.Data.RunningTotal)
			data := &twPosition{
				t:                      time.Unix(0, td.Time),
				position:               position,
				previousEpochsIdx:      int(td.Data.PreviousEpochsIndex),
				currentEpochTWPosition: current,
				previousEpochs:         make([]num.Decimal, 0, maxWindowSize),
			}
			for _, v := range td.Data.PreviousEpochs {
				d, _ := num.DecimalFromString(v)
				data.previousEpochs = append(data.previousEpochs, d)
			}
			mft.timeWeightedPosition[td.Party] = data
		}
	}

	if len(tracker.ReturnsData) > 0 {
		for _, rd := range tracker.ReturnsData {
			runningTotal, _ := num.DecimalFromString(rd.Data.RunningTotal)
			data := &m2mData{
				previousEpochsIdx: int(rd.Data.PreviousEpochsIndex),
				previousEpochs:    make([]num.Decimal, 0, maxWindowSize),
				runningTotal:      runningTotal,
			}
			for _, v := range rd.Data.PreviousEpochs {
				d, _ := num.DecimalFromString(v)
				data.previousEpochs = append(data.previousEpochs, d)
			}
			mft.partyM2M[rd.Party] = data
		}
	}

	loadFeesHistory(tracker.MakerFeesPaidHistory, mft.makerFeesPaid)
	loadFeesHistory(tracker.MakerFeesReceivedHistory, mft.makerFeesReceived)
	loadFeesHistory(tracker.LpFeesHistory, mft.lpFees)

	if tracker.TotalLpFees != nil {
		mft.totalLpFees = loadTotalFee(tracker.TotalLpFees)
	}
	if tracker.TotalMakerFeesPaid != nil {
		mft.totalMakerFeesPaid = loadTotalFee(tracker.TotalMakerFeesPaid)
	}
	if tracker.TotalMakerFeesReceived != nil {
		mft.totalMakerFeesReceived = loadTotalFee(tracker.TotalMakerFeesReceived)
	}

	mft.asset = tracker.Asset
	return mft
}

func loadTotalFee(cpFeesData *checkpoint.DataWithHistory) *feeData {
	runningTotal, _ := num.UintFromString(cpFeesData.RunningTotal, 10)
	fd := &feeData{
		runningTotal:      runningTotal,
		previousEpochsIdx: int(cpFeesData.GetPreviousEpochsIndex()),
		previousEpochs:    make([]*num.Uint, maxWindowSize),
	}
	for i, v := range cpFeesData.PreviousEpochs {
		if len(v) > 0 {
			d, _ := num.UintFromString(v, 10)
			fd.previousEpochs[i] = d
		} else {
			fd.previousEpochs[i] = nil
		}
	}

	return fd
}

func loadFeesHistory(cpFeesData []*checkpoint.PartyFeeData, feesData map[string]*feeData) {
	for _, pfd := range cpFeesData {
		runningTotal, _ := num.UintFromString(pfd.Data.RunningTotal, 10)
		data := &feeData{
			runningTotal:      runningTotal,
			previousEpochsIdx: int(pfd.Data.PreviousEpochsIndex),
			previousEpochs:    make([]*num.Uint, 0, maxWindowSize),
		}
		for _, v := range pfd.Data.PreviousEpochs {
			if v == "" {
				data.previousEpochs = append(data.previousEpochs, nil)
			} else {
				d, _ := num.UintFromString(v, 10)
				data.previousEpochs = append(data.previousEpochs, d)
			}
		}
		feesData[pfd.Party] = data
	}
}

func (mat *MarketActivityTracker) restore(tracker *snapshot.MarketTracker) {
	for _, data := range tracker.MarketActivity {
		if _, ok := mat.assetToMarketTrackers[data.Asset]; !ok {
			mat.assetToMarketTrackers[data.Asset] = map[string]*marketTracker{}
		}
		mat.assetToMarketTrackers[data.Asset][data.Market] = marketTrackerFromProto(data)
	}
	for _, tnv := range tracker.TakerNotionalVolume {
		volume, _ := num.UintFromString(tnv.Volume, 10)
		mat.partyTakerNotionalVolume[tnv.Party] = volume
	}
}

// onEpochRestore is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) onEpochRestore(_ context.Context, epoch types.Epoch) {
	mat.currentEpoch = epoch.Seq
	mat.epochStartTime = epoch.StartTime
}
