package execution

import (
	"context"
	"errors"
	"sort"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	key                        = (&types.PayloadMarketActivityTracker{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for market activity tracker snapshot")
	hashKeys                   = []string{key}
)

type snapshotState struct {
	changed    bool
	hash       []byte
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

func (mat *MarketActivityTracker) serialise() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadMarketActivityTracker{
			MarketActivityData: mat.serialiseFeesTracker(),
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form and hash of the given key.
func (mat *MarketActivityTracker) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != key {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !mat.ss.changed {
		return mat.ss.serialised, mat.ss.hash, nil
	}

	data, err := mat.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	mat.ss.serialised = data
	mat.ss.hash = hash
	mat.ss.changed = false
	return data, hash, nil
}

func (mat *MarketActivityTracker) GetHash(k string) ([]byte, error) {
	_, hash, err := mat.getSerialisedAndHash(k)
	return hash, err
}

func (mat *MarketActivityTracker) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := mat.getSerialisedAndHash(k)
	return state, nil, err
}

func (mat *MarketActivityTracker) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if mat.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadMarketActivityTracker:
		return nil, mat.restore(ctx, pl.MarketActivityData)
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

func (mat *MarketActivityTracker) restore(ctx context.Context, data *snapshot.MarketTracker) error {
	for _, data := range data.MarketActivity {
		mat.marketToTracker[data.Market] = marketTrackerFromProto(data)
	}
	mat.ss.changed = true
	return nil
}

// onEpochRestore is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) onEpochRestore(_ context.Context, epoch types.Epoch) {
	mat.currentEpoch = epoch.Seq
}
