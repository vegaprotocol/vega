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

package execution

import (
	"context"
	"errors"
	"sort"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"

	"code.vegaprotocol.io/vega/core/libs/proto"
)

var (
	key                        = (&types.PayloadMarketActivityTracker{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for market activity tracker snapshot")
	hashKeys                   = []string{key}
)

type snapshotState struct {
	changed    bool
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
	return &checkpoint.MarketActivityTracker{
		Asset:         mt.asset,
		Market:        market,
		MakerFees:     marketFeesToProto(mt.makerFees),
		TakerFees:     marketFeesToProto(mt.takerFees),
		LpFees:        marketFeesToProto(mt.lpFees),
		ValueTraded:   mt.valueTraded.String(),
		Proposer:      mt.proposer,
		BonusPaid:     mt.proposersPaid,
		ReadyToDelete: mt.readyToDelete,
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

	if !mat.HasChanged(k) {
		return mat.ss.serialised, nil
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
	mat.ss.changed = false
	return data, nil
}

func (mat *MarketActivityTracker) HasChanged(k string) bool {
	// return mat.ss.changed
	return true
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
		mat.restore(ctx, pl.MarketActivityData)
		var err error
		mat.ss.changed = false
		mat.ss.serialised, err = proto.Marshal(p.IntoProto())
		return nil, err
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func marketTrackerFromProto(data *checkpoint.MarketActivityTracker) *marketTracker {
	valueTrades, _ := num.UintFromString(data.ValueTraded, 10)
	mft := &marketTracker{
		makerFees:      map[string]*num.Uint{},
		takerFees:      map[string]*num.Uint{},
		lpFees:         map[string]*num.Uint{},
		totalMakerFees: num.Zero(),
		totalTakerFees: num.Zero(),
		totalLPFees:    num.Zero(),
		valueTraded:    valueTrades,
		proposer:       data.Proposer,
		proposersPaid:  data.BonusPaid,
		asset:          data.Asset,
		readyToDelete:  data.ReadyToDelete,
	}
	for _, mf := range data.MakerFees {
		mft.makerFees[mf.Party], _ = num.UintFromString(mf.Fee, 10)
		mft.totalMakerFees.AddSum(mft.makerFees[mf.Party])
	}
	for _, tf := range data.TakerFees {
		mft.takerFees[tf.Party], _ = num.UintFromString(tf.Fee, 10)
		mft.totalTakerFees.AddSum(mft.takerFees[tf.Party])
	}
	for _, lp := range data.LpFees {
		mft.lpFees[lp.Party], _ = num.UintFromString(lp.Fee, 10)
		mft.totalLPFees.AddSum(mft.lpFees[lp.Party])
	}
	mft.asset = data.Asset
	return mft
}

func (mat *MarketActivityTracker) restore(ctx context.Context, data *snapshot.MarketTracker) {
	for _, data := range data.MarketActivity {
		mat.marketToTracker[data.Market] = marketTrackerFromProto(data)
	}
}

// onEpochRestore is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) onEpochRestore(_ context.Context, epoch types.Epoch) {
	mat.currentEpoch = epoch.Seq
}
