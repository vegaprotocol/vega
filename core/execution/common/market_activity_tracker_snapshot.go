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
		Asset:             mt.asset,
		Market:            market,
		MakerFeesReceived: marketFeesToProto(mt.makerFeesReceived),
		MakerFeesPaid:     marketFeesToProto(mt.makerFeesPaid),
		LpFees:            marketFeesToProto(mt.lpFees),
		ValueTraded:       mt.valueTraded.String(),
		Proposer:          mt.proposer,
		BonusPaid:         paid,
		ReadyToDelete:     mt.readyToDelete,
	}
}

func (mat *MarketActivityTracker) serialiseFeesTracker() *snapshot.MarketTracker {
	markets := make([]string, 0, len(mat.marketToTracker))
	for k := range mat.marketToTracker {
		markets = append(markets, k)
	}
	sort.Strings(markets)

	marketActivity := make([]*checkpoint.MarketActivityTracker, 0, len(markets))
	for _, market := range markets {
		marketActivity = append(marketActivity, mat.marketToTracker[market].IntoProto(market))
	}

	return &snapshot.MarketTracker{
		MarketActivity: marketActivity,
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

func marketTrackerFromProto(data *checkpoint.MarketActivityTracker) *marketTracker {
	valueTrades, _ := num.UintFromString(data.ValueTraded, 10)
	mft := &marketTracker{
		makerFeesReceived:      map[string]*num.Uint{},
		makerFeesPaid:          map[string]*num.Uint{},
		lpFees:                 map[string]*num.Uint{},
		totalMakerFeesReceived: num.UintZero(),
		totalMakerFeesPaid:     num.UintZero(),
		totalLPFees:            num.UintZero(),
		valueTraded:            valueTrades,
		proposer:               data.Proposer,
		proposersPaid:          map[string]struct{}{},
		asset:                  data.Asset,
		readyToDelete:          data.ReadyToDelete,
	}

	for _, bpfpa := range data.BonusPaid {
		mft.proposersPaid[bpfpa] = struct{}{}
	}

	for _, mf := range data.MakerFeesReceived {
		mft.makerFeesReceived[mf.Party], _ = num.UintFromString(mf.Fee, 10)
		mft.totalMakerFeesReceived.AddSum(mft.makerFeesReceived[mf.Party])
	}
	for _, tf := range data.MakerFeesPaid {
		mft.makerFeesPaid[tf.Party], _ = num.UintFromString(tf.Fee, 10)
		mft.totalMakerFeesPaid.AddSum(mft.makerFeesPaid[tf.Party])
	}
	for _, lp := range data.LpFees {
		mft.lpFees[lp.Party], _ = num.UintFromString(lp.Fee, 10)
		mft.totalLPFees.AddSum(mft.lpFees[lp.Party])
	}
	mft.asset = data.Asset
	return mft
}

func (mat *MarketActivityTracker) restore(data *snapshot.MarketTracker) {
	for _, data := range data.MarketActivity {
		mat.marketToTracker[data.Market] = marketTrackerFromProto(data)
	}
}

// onEpochRestore is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) onEpochRestore(_ context.Context, epoch types.Epoch) {
	mat.currentEpoch = epoch.Seq
}
