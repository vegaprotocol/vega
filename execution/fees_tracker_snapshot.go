package execution

import (
	"context"
	"errors"
	"sort"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	key                        = (&types.PayloadFeeTracker{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for fee tracker snapshot")
	hashKeys                   = []string{key}
)

type snapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (f *FeesTracker) Namespace() types.SnapshotNamespace {
	return types.FeeTrackerSnapshot
}

func (f *FeesTracker) Keys() []string {
	return hashKeys
}

func (f *FeesTracker) Stopped() bool {
	return false
}

func marketFeesToProto(partyFees map[string]*num.Uint) []*snapshot.PartyFees {
	parties := make([]string, 0, len(partyFees))
	for k := range partyFees {
		parties = append(parties, k)
	}
	sort.Strings(parties)
	pf := make([]*snapshot.PartyFees, 0, len(parties))
	for _, party := range parties {
		pf = append(pf, &snapshot.PartyFees{Party: party, Fee: partyFees[party].String()})
	}
	return pf
}

func (mft *marketFeesTracker) IntoProto(market string) *snapshot.MarketFees {
	return &snapshot.MarketFees{
		Market:    market,
		MakerFees: marketFeesToProto(mft.makerFees),
		TakerFees: marketFeesToProto(mft.takerFees),
		LpFees:    marketFeesToProto(mft.lpFees),
	}
}

func (f *FeesTracker) serialiseFeesTracker() *snapshot.FeesTracker {
	markets := make([]string, 0, len(f.marketToTracker))
	for k := range f.marketToTracker {
		markets = append(markets, k)
	}
	sort.Strings(markets)

	marketFees := make([]*snapshot.MarketFees, 0, len(markets))
	for _, market := range markets {
		marketFees = append(marketFees, f.marketToTracker[market].IntoProto(market))
	}

	return &snapshot.FeesTracker{
		MarketFees: marketFees,
	}
}

func (f *FeesTracker) serialise() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadFeeTracker{
			FeeTrackerData: f.serialiseFeesTracker(),
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form and hash of the given key.
func (f *FeesTracker) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != key {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !f.ss.changed {
		return f.ss.serialised, f.ss.hash, nil
	}

	data, err := f.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	f.ss.serialised = data
	f.ss.hash = hash
	f.ss.changed = false
	return data, hash, nil
}

func (f *FeesTracker) GetHash(k string) ([]byte, error) {
	_, hash, err := f.getSerialisedAndHash(k)
	return hash, err
}

func (f *FeesTracker) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := f.getSerialisedAndHash(k)
	return state, nil, err
}

func (f *FeesTracker) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if f.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadFeeTracker:
		return nil, f.restore(ctx, pl.FeeTrackerData)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func marketFeesTrackerFromProto(data *snapshot.MarketFees) *marketFeesTracker {
	mft := &marketFeesTracker{
		makerFees: map[string]*num.Uint{},
		takerFees: map[string]*num.Uint{},
		lpFees:    map[string]*num.Uint{},
	}
	for _, mf := range data.MakerFees {
		mft.makerFees[mf.Party], _ = num.UintFromString(mf.Fee, 10)
	}
	for _, tf := range data.TakerFees {
		mft.takerFees[tf.Party], _ = num.UintFromString(tf.Fee, 10)
	}
	for _, lp := range data.LpFees {
		mft.lpFees[lp.Party], _ = num.UintFromString(lp.Fee, 10)
	}
	return mft
}

func (f *FeesTracker) restore(ctx context.Context, data *snapshot.FeesTracker) error {
	for _, data := range data.MarketFees {
		f.marketToTracker[data.Market] = marketFeesTrackerFromProto(data)
	}
	f.ss.changed = true
	return nil
}

// onEpochEvent is called when the state of the epoch changes, we only care about new epochs starting.
func (f *FeesTracker) onEpochRestore(_ context.Context, epoch types.Epoch) {
	f.currentEpoch = epoch.Seq
}
