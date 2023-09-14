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

func returnsDataToProto(partyM2MData map[string]num.Decimal) []*checkpoint.ReturnsData {
	parties := make([]string, 0, len(partyM2MData))
	for k := range partyM2MData {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	data := make([]*checkpoint.ReturnsData, 0, len(parties))
	for _, party := range parties {
		rd := &checkpoint.ReturnsData{Party: party}
		rd.Return, _ = partyM2MData[party].MarshalBinary()
		data = append(data, rd)
	}
	return data
}

func epochReturnDataToProto(epochData []map[string]num.Decimal) []*checkpoint.EpochReturnsData {
	ret := make([]*checkpoint.EpochReturnsData, 0, len(epochData))
	for _, v := range epochData {
		ed := make([]*checkpoint.ReturnsData, 0, len(v))
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, party := range keys {
			retData := &checkpoint.ReturnsData{
				Party: party,
			}
			retData.Return, _ = v[party].MarshalBinary()
			ed = append(ed, retData)
		}
		ret = append(ret, &checkpoint.EpochReturnsData{Returns: ed})
	}
	return ret
}

func timeWeightedNotionalToProto(twNotional map[string]*twNotional) []*checkpoint.TWNotionalData {
	parties := make([]string, 0, len(twNotional))
	for k := range twNotional {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	data := make([]*checkpoint.TWNotionalData, 0, len(parties))
	for _, party := range parties {
		pd := twNotional[party]
		pdProto := &checkpoint.TWNotionalData{
			Party: party,
			Time:  pd.t.UnixNano(),
		}
		b := pd.notional.Bytes()
		pdProto.Notional = b[:]
		twb := pd.currentEpochTWNotional.Bytes()
		pdProto.TwNotional = twb[:]
		data = append(data, pdProto)
	}
	return data
}

func timeWeightedNotionalHistoryToProto(partyNotionalHistory []map[string]*num.Uint) []*checkpoint.EpochTimeWeightedNotionalData {
	ret := make([]*checkpoint.EpochTimeWeightedNotionalData, 0, len(partyNotionalHistory))
	for _, v := range partyNotionalHistory {
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		epochData := &checkpoint.EpochTimeWeightedNotionalData{PartyTimeWeightedNotionals: make([]*checkpoint.PartyTimeWeightedNotional, 0, len(keys))}
		for _, party := range keys {
			partyData := &checkpoint.PartyTimeWeightedNotional{Party: party}
			b := v[party].Bytes()
			partyData.TwNotional = b[:]
			epochData.PartyTimeWeightedNotionals = append(epochData.PartyTimeWeightedNotionals, partyData)
		}
		ret = append(ret, epochData)
	}
	return ret
}

func timeWeightedPositionHistoryToProto(partyPositionsHistory []map[string]uint64) []*checkpoint.EpochTimeWeightPositionData {
	ret := make([]*checkpoint.EpochTimeWeightPositionData, 0, len(partyPositionsHistory))
	for _, v := range partyPositionsHistory {
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		epochData := &checkpoint.EpochTimeWeightPositionData{PartyTimeWeightedPositions: make([]*checkpoint.PartyTimeWeightedPosition, 0, len(keys))}
		for _, party := range keys {
			epochData.PartyTimeWeightedPositions = append(epochData.PartyTimeWeightedPositions, &checkpoint.PartyTimeWeightedPosition{Party: party, TwPosition: v[party]})
		}
		ret = append(ret, epochData)
	}
	return ret
}

func timeWeightedPositionToProto(partyPositions map[string]*twPosition) []*checkpoint.TWPositionData {
	parties := make([]string, 0, len(partyPositions))
	for k := range partyPositions {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	data := make([]*checkpoint.TWPositionData, 0, len(parties))
	for _, party := range parties {
		pd := partyPositions[party]
		pdProto := &checkpoint.TWPositionData{
			Party:      party,
			Time:       pd.t.UnixNano(),
			Position:   pd.position,
			TwPosition: pd.currentEpochTWPosition,
		}
		data = append(data, pdProto)
	}
	return data
}

func takerNotionalToProto(takerNotional map[string]*num.Uint) []*checkpoint.TakerNotionalVolume {
	ret := make([]*checkpoint.TakerNotionalVolume, 0, len(takerNotional))
	for k, u := range takerNotional {
		var b []byte
		if u != nil {
			bb := u.Bytes()
			b = bb[:]
		}
		ret = append(ret, &checkpoint.TakerNotionalVolume{Party: k, Volume: b})
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Party < ret[j].Party
	})
	return ret
}

func marketFeesHistoryToProto(feeHistory []map[string]*num.Uint) []*checkpoint.EpochPartyFees {
	data := make([]*checkpoint.EpochPartyFees, 0, len(feeHistory))
	for _, v := range feeHistory {
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		partyFees := &checkpoint.EpochPartyFees{PartyFees: make([]*checkpoint.PartyFeesHistory, 0, len(keys))}
		for _, party := range keys {
			pfh := &checkpoint.PartyFeesHistory{Party: party}
			b := v[party].Bytes()
			pfh.Fee = b[:]
			partyFees.PartyFees = append(partyFees.PartyFees, pfh)
		}
		data = append(data, partyFees)
	}
	return data
}

func marketFeesToProto(partyFees map[string]*num.Uint) []*checkpoint.PartyFees {
	parties := make([]string, 0, len(partyFees))
	for k := range partyFees {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	pf := make([]*checkpoint.PartyFees, 0, len(parties))
	for _, party := range parties {
		pf = append(pf, &checkpoint.PartyFees{Party: party, Fee: partyFees[party].String()})
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
		Market:                          market,
		Asset:                           mt.asset,
		MakerFeesReceived:               marketFeesToProto(mt.makerFeesReceived),
		MakerFeesPaid:                   marketFeesToProto(mt.makerFeesPaid),
		LpFees:                          marketFeesToProto(mt.lpFees),
		Proposer:                        mt.proposer,
		BonusPaid:                       paid,
		ValueTraded:                     mt.valueTraded.String(),
		ReadyToDelete:                   mt.readyToDelete,
		TimeWeightedPosition:            timeWeightedPositionToProto(mt.twPosition),
		TimeWeightedNotional:            timeWeightedNotionalToProto(mt.twNotional),
		ReturnsData:                     returnsDataToProto(mt.partyM2M),
		MakerFeesReceivedHistory:        marketFeesHistoryToProto(mt.epochMakerFeesReceived),
		MakerFeesPaidHistory:            marketFeesHistoryToProto(mt.epochMakerFeesPaid),
		LpFeesHistory:                   marketFeesHistoryToProto(mt.epochLpFees),
		TimeWeightedPositionDataHistory: timeWeightedPositionHistoryToProto(mt.epochTimeWeightedPosition),
		TimeWeightedNotionalDataHistory: timeWeightedNotionalHistoryToProto(mt.epochTimeWeightedNotional),
		ReturnsDataHistory:              epochReturnDataToProto(mt.epochPartyM2M),
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
		asset:                  tracker.Asset,
		proposer:               tracker.Proposer,
		proposersPaid:          map[string]struct{}{},
		readyToDelete:          tracker.ReadyToDelete,
		valueTraded:            valueTrades,
		makerFeesReceived:      map[string]*num.Uint{},
		makerFeesPaid:          map[string]*num.Uint{},
		lpFees:                 map[string]*num.Uint{},
		totalMakerFeesReceived: num.UintZero(),
		totalMakerFeesPaid:     num.UintZero(),
		totalLpFees:            num.UintZero(),
		twPosition:             map[string]*twPosition{},
		partyM2M:               map[string]num.Decimal{},
		twNotional:             map[string]*twNotional{},

		epochTotalMakerFeesReceived: []*num.Uint{},
		epochTotalMakerFeesPaid:     []*num.Uint{},
		epochTotalLpFees:            []*num.Uint{},
		epochMakerFeesReceived:      []map[string]*num.Uint{},
		epochMakerFeesPaid:          []map[string]*num.Uint{},
		epochLpFees:                 []map[string]*num.Uint{},
		epochPartyM2M:               []map[string]num.Decimal{},
		epochTimeWeightedPosition:   []map[string]uint64{},
		epochTimeWeightedNotional:   []map[string]*num.Uint{},
		allPartiesCache:             map[string]struct{}{},
	}

	for _, bpfpa := range tracker.BonusPaid {
		mft.proposersPaid[bpfpa] = struct{}{}
	}

	if len(tracker.MakerFeesReceived) > 0 {
		total := num.UintZero()
		for _, mf := range tracker.MakerFeesReceived {
			fee, _ := num.UintFromString(mf.Fee, 10)
			total.AddSum(fee)
			mft.makerFeesReceived[mf.Party] = fee
			mft.allPartiesCache[mf.Party] = struct{}{}
		}
		mft.totalMakerFeesReceived = total
	}

	if len(tracker.MakerFeesPaid) > 0 {
		total := num.UintZero()
		for _, mf := range tracker.MakerFeesPaid {
			fee, _ := num.UintFromString(mf.Fee, 10)
			total.AddSum(fee)
			mft.makerFeesPaid[mf.Party] = fee
			mft.allPartiesCache[mf.Party] = struct{}{}
		}
		mft.totalMakerFeesPaid = total
	}

	if len(tracker.LpFees) > 0 {
		total := num.UintZero()
		for _, mf := range tracker.LpFees {
			fee, _ := num.UintFromString(mf.Fee, 10)
			total.AddSum(fee)
			mft.lpFees[mf.Party] = fee
			mft.allPartiesCache[mf.Party] = struct{}{}
		}
		mft.totalLpFees = total
	}

	if len(tracker.TimeWeightedPosition) > 0 {
		for _, tp := range tracker.TimeWeightedPosition {
			mft.twPosition[tp.Party] = &twPosition{
				position:               tp.Position,
				t:                      time.Unix(0, tp.Time),
				currentEpochTWPosition: tp.TwPosition,
			}
			mft.allPartiesCache[tp.Party] = struct{}{}
		}
	}

	if len(tracker.TimeWeightedNotional) > 0 {
		for _, tn := range tracker.TimeWeightedNotional {
			mft.twNotional[tn.Party] = &twNotional{
				notional:               num.UintFromBytes(tn.Notional),
				t:                      time.Unix(0, tn.Time),
				currentEpochTWNotional: num.UintFromBytes(tn.TwNotional),
			}
			mft.allPartiesCache[tn.Party] = struct{}{}
		}
	}

	if len(tracker.ReturnsData) > 0 {
		for _, rd := range tracker.ReturnsData {
			ret, _ := num.UnmarshalBinaryDecimal(rd.Return)
			mft.partyM2M[rd.Party] = ret
			mft.allPartiesCache[rd.Party] = struct{}{}
		}
	}

	mft.epochMakerFeesPaid = loadFeesHistory(tracker.MakerFeesPaidHistory, mft.allPartiesCache)
	mft.epochMakerFeesReceived = loadFeesHistory(tracker.MakerFeesReceivedHistory, mft.allPartiesCache)
	mft.epochLpFees = loadFeesHistory(tracker.LpFeesHistory, mft.allPartiesCache)

	mft.epochTotalMakerFeesPaid = updateTotalHistory(mft.epochMakerFeesPaid)
	mft.epochTotalMakerFeesReceived = updateTotalHistory(mft.epochMakerFeesReceived)
	mft.epochTotalLpFees = updateTotalHistory(mft.epochLpFees)

	if len(tracker.TimeWeightedPositionDataHistory) > 0 {
		for _, etwnd := range tracker.TimeWeightedPositionDataHistory {
			m := make(map[string]uint64, len(etwnd.PartyTimeWeightedPositions))
			for _, partyPositions := range etwnd.PartyTimeWeightedPositions {
				m[partyPositions.Party] = partyPositions.TwPosition
				mft.allPartiesCache[partyPositions.Party] = struct{}{}
			}
			mft.epochTimeWeightedPosition = append(mft.epochTimeWeightedPosition, m)
		}
	}

	if len(tracker.TimeWeightedNotionalDataHistory) > 0 {
		for _, etwnd := range tracker.TimeWeightedNotionalDataHistory {
			m := make(map[string]*num.Uint, len(etwnd.PartyTimeWeightedNotionals))
			for _, partyNotionals := range etwnd.PartyTimeWeightedNotionals {
				m[partyNotionals.Party] = num.UintFromBytes(partyNotionals.TwNotional)
				mft.allPartiesCache[partyNotionals.Party] = struct{}{}
			}
			mft.epochTimeWeightedNotional = append(mft.epochTimeWeightedNotional, m)
		}
	}

	if len(tracker.ReturnsDataHistory) > 0 {
		for _, erd := range tracker.ReturnsDataHistory {
			returns := make(map[string]num.Decimal, len(erd.Returns))
			for _, rd := range erd.Returns {
				ret, _ := num.UnmarshalBinaryDecimal(rd.Return)
				returns[rd.Party] = ret
				mft.allPartiesCache[rd.Party] = struct{}{}
			}
			mft.epochPartyM2M = append(mft.epochPartyM2M, returns)
		}
	}

	mft.asset = tracker.Asset
	return mft
}

func updateTotalHistory(data []map[string]*num.Uint) []*num.Uint {
	ret := make([]*num.Uint, 0, len(data))
	for _, v := range data {
		total := num.UintZero()
		for _, u := range v {
			total.AddSum(u)
		}
		ret = append(ret, total)
	}
	return ret
}

func loadFeesHistory(cpFeesData []*checkpoint.EpochPartyFees, allParties map[string]struct{}) []map[string]*num.Uint {
	feeData := make([]map[string]*num.Uint, 0, len(cpFeesData))
	for _, pfd := range cpFeesData {
		epochTotal := num.UintZero()
		m := make(map[string]*num.Uint, len(pfd.PartyFees))
		for _, pfh := range pfd.PartyFees {
			fee := num.UintFromBytes(pfh.Fee)
			m[pfh.Party] = fee
			epochTotal.AddSum(fee)
			allParties[pfh.Party] = struct{}{}
		}
		feeData = append(feeData, m)
	}
	return feeData
}

func (mat *MarketActivityTracker) restore(tracker *snapshot.MarketTracker) {
	for _, data := range tracker.MarketActivity {
		if _, ok := mat.assetToMarketTrackers[data.Asset]; !ok {
			mat.assetToMarketTrackers[data.Asset] = map[string]*marketTracker{}
		}
		mat.assetToMarketTrackers[data.Asset][data.Market] = marketTrackerFromProto(data)
	}
	for _, tnv := range tracker.TakerNotionalVolume {
		if len(tnv.Volume) > 0 {
			mat.partyTakerNotionalVolume[tnv.Party] = num.UintFromBytes(tnv.Volume)
		}
	}
}

// onEpochRestore is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) onEpochRestore(_ context.Context, epoch types.Epoch) {
	mat.currentEpoch = epoch.Seq
	mat.epochStartTime = epoch.StartTime
}
