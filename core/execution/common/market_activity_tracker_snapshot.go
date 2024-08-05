// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
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

func returnsDataToProto(returnsData map[string]num.Decimal) []*checkpoint.ReturnsData {
	parties := make([]string, 0, len(returnsData))
	for k := range returnsData {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	data := make([]*checkpoint.ReturnsData, 0, len(parties))
	for _, party := range parties {
		rd := &checkpoint.ReturnsData{Party: party}
		rd.Return, _ = returnsData[party].MarshalBinary()
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

func epochTakerFeesToProto(epochData []map[string]map[string]map[string]*num.Uint) []*checkpoint.EpochPartyTakerFees {
	ret := make([]*checkpoint.EpochPartyTakerFees, 0, len(epochData))
	for _, epoch := range epochData {
		ed := []*checkpoint.AssetMarketPartyTakerFees{}
		assets := make([]string, 0, len(epoch))
		for k := range epoch {
			assets = append(assets, k)
		}
		sort.Strings(assets)
		for _, asset := range assets {
			assetData := epoch[asset]
			markets := make([]string, 0, len(assetData))
			for market := range assetData {
				markets = append(markets, market)
			}
			sort.Strings(markets)
			for _, market := range markets {
				takerFees := assetData[market]
				parties := make([]string, 0, len(takerFees))
				for party := range takerFees {
					parties = append(parties, party)
				}
				sort.Strings(parties)
				partyFees := make([]*checkpoint.PartyTakerFees, 0, len(parties))
				for _, party := range parties {
					fee := takerFees[party].Bytes()
					partyFees = append(partyFees, &checkpoint.PartyTakerFees{
						Party:     party,
						TakerFees: fee[:],
					})
				}
				ed = append(ed, &checkpoint.AssetMarketPartyTakerFees{
					Asset:     asset,
					Market:    market,
					TakerFees: partyFees,
				})
			}
		}
		ret = append(ret, &checkpoint.EpochPartyTakerFees{EpochPartyTakerFeesPaid: ed})
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

func marketToPartyTakerNotionalToProto(stats map[string]map[string]*num.Uint) []*checkpoint.MarketToPartyTakerNotionalVolume {
	ret := make([]*checkpoint.MarketToPartyTakerNotionalVolume, 0, len(stats))
	for marketID, partiesStats := range stats {
		ret = append(ret, &checkpoint.MarketToPartyTakerNotionalVolume{
			Market:              marketID,
			TakerNotionalVolume: takerNotionalToProto(partiesStats),
		})
	}

	slices.SortStableFunc(ret, func(a, b *checkpoint.MarketToPartyTakerNotionalVolume) int {
		return strings.Compare(a.Market, b.Market)
	})

	return ret
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

	ammParties := maps.Keys(mt.ammPartiesCache)
	sort.Strings(ammParties)

	return &checkpoint.MarketActivityTracker{
		Market:                          market,
		Asset:                           mt.asset,
		MakerFeesReceived:               marketFeesToProto(mt.makerFeesReceived),
		MakerFeesPaid:                   marketFeesToProto(mt.makerFeesPaid),
		LpFees:                          marketFeesToProto(mt.lpFees),
		InfraFees:                       marketFeesToProto(mt.infraFees),
		LpPaidFees:                      marketFeesToProto(mt.lpPaidFees),
		BuyBackFees:                     marketFeesToProto(mt.buybackFeesPaid),
		TreasuryFees:                    marketFeesToProto(mt.treasuryFeesPaid),
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
		RealisedReturns:                 returnsDataToProto(mt.partyRealisedReturn),
		RealisedReturnsHistory:          epochReturnDataToProto(mt.epochPartyRealisedReturn),
		AmmParties:                      ammParties,
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
		MarketActivity:                   marketActivity,
		TakerNotionalVolume:              takerNotionalToProto(mat.partyTakerNotionalVolume),
		MarketToPartyTakerNotionalVolume: marketToPartyTakerNotionalToProto(mat.marketToPartyTakerNotionalVolume),
		EpochTakerFees:                   epochTakerFeesToProto(mat.takerFeesPaidInEpoch),
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

func (mat *MarketActivityTracker) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
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
		buybackFeesPaid:        map[string]*num.Uint{},
		treasuryFeesPaid:       map[string]*num.Uint{},
		infraFees:              map[string]*num.Uint{},
		lpPaidFees:             map[string]*num.Uint{},
		totalMakerFeesReceived: num.UintZero(),
		totalMakerFeesPaid:     num.UintZero(),
		totalLpFees:            num.UintZero(),
		twPosition:             map[string]*twPosition{},
		partyM2M:               map[string]num.Decimal{},
		partyRealisedReturn:    map[string]num.Decimal{},
		twNotional:             map[string]*twNotional{},

		epochTotalMakerFeesReceived: []*num.Uint{},
		epochTotalMakerFeesPaid:     []*num.Uint{},
		epochTotalLpFees:            []*num.Uint{},
		epochMakerFeesReceived:      []map[string]*num.Uint{},
		epochMakerFeesPaid:          []map[string]*num.Uint{},
		epochLpFees:                 []map[string]*num.Uint{},
		epochPartyM2M:               []map[string]num.Decimal{},
		epochPartyRealisedReturn:    []map[string]num.Decimal{},
		epochTimeWeightedPosition:   []map[string]uint64{},
		epochTimeWeightedNotional:   []map[string]*num.Uint{},
		allPartiesCache:             map[string]struct{}{},
		ammPartiesCache:             map[string]struct{}{},
	}

	for _, party := range tracker.AmmParties {
		mft.ammPartiesCache[party] = struct{}{}
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

	if len(tracker.InfraFees) > 0 {
		for _, mf := range tracker.InfraFees {
			fee, _ := num.UintFromString(mf.Fee, 10)
			mft.infraFees[mf.Party] = fee
			mft.allPartiesCache[mf.Party] = struct{}{}
		}
	}

	if len(tracker.BuyBackFees) > 0 {
		for _, mf := range tracker.BuyBackFees {
			fee, _ := num.UintFromString(mf.Fee, 10)
			mft.buybackFeesPaid[mf.Party] = fee
			mft.allPartiesCache[mf.Party] = struct{}{}
		}
	}

	if len(tracker.TreasuryFees) > 0 {
		for _, mf := range tracker.TreasuryFees {
			fee, _ := num.UintFromString(mf.Fee, 10)
			mft.treasuryFeesPaid[mf.Party] = fee
			mft.allPartiesCache[mf.Party] = struct{}{}
		}
	}

	if len(tracker.LpPaidFees) > 0 {
		for _, mf := range tracker.LpPaidFees {
			fee, _ := num.UintFromString(mf.Fee, 10)
			mft.lpPaidFees[mf.Party] = fee
			mft.allPartiesCache[mf.Party] = struct{}{}
		}
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

	if len(tracker.RealisedReturns) > 0 {
		for _, rd := range tracker.RealisedReturns {
			ret, _ := num.UnmarshalBinaryDecimal(rd.Return)
			mft.partyRealisedReturn[rd.Party] = ret
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

	if len(tracker.RealisedReturnsHistory) > 0 {
		for _, erd := range tracker.RealisedReturnsHistory {
			returns := make(map[string]num.Decimal, len(erd.Returns))
			for _, rd := range erd.Returns {
				ret, _ := num.UnmarshalBinaryDecimal(rd.Return)
				returns[rd.Party] = ret
				mft.allPartiesCache[rd.Party] = struct{}{}
			}
			mft.epochPartyRealisedReturn = append(mft.epochPartyRealisedReturn, returns)
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
	for _, marketToPartyStats := range tracker.MarketToPartyTakerNotionalVolume {
		mat.marketToPartyTakerNotionalVolume[marketToPartyStats.Market] = map[string]*num.Uint{}
		for _, partyStats := range marketToPartyStats.TakerNotionalVolume {
			if len(partyStats.Volume) > 0 {
				mat.marketToPartyTakerNotionalVolume[marketToPartyStats.Market][partyStats.Party] = num.UintFromBytes(partyStats.Volume)
			}
		}
	}
	if tracker.EpochTakerFees != nil {
		for _, epochData := range tracker.EpochTakerFees {
			epochMap := map[string]map[string]map[string]*num.Uint{}
			for _, assetMarketParty := range epochData.EpochPartyTakerFeesPaid {
				if _, ok := epochMap[assetMarketParty.Asset]; !ok {
					epochMap[assetMarketParty.Asset] = map[string]map[string]*num.Uint{}
				}
				if _, ok := epochMap[assetMarketParty.Asset][assetMarketParty.Market]; !ok {
					epochMap[assetMarketParty.Asset][assetMarketParty.Market] = map[string]*num.Uint{}
				}
				for _, tf := range assetMarketParty.TakerFees {
					epochMap[assetMarketParty.Asset][assetMarketParty.Market][tf.Party] = num.UintFromBytes(tf.TakerFees)
				}
			}
			mat.takerFeesPaidInEpoch = append(mat.takerFeesPaidInEpoch, epochMap)
		}
	}
}

// OnEpochRestore is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) OnEpochRestore(_ context.Context, epoch types.Epoch) {
	mat.currentEpoch = epoch.Seq
	mat.epochStartTime = epoch.StartTime
}
