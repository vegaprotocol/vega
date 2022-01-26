package execution

import (
	"context"
	"errors"
	"sort"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
)

var (
	marketTrackerkey                           = (&types.PayloadMarketTracker{}).Key()
	ErrSnapshotKeyDoesNotExistForMarketTracker = errors.New("unknown key for market tracker snapshot")
	marketTrackerHashKeys                      = []string{marketTrackerkey}
)

type marketTrackerSnapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (m *MarketTracker) Namespace() types.SnapshotNamespace {
	return types.MarketTrackerSnapshot
}

func (m *MarketTracker) Keys() []string {
	return marketTrackerHashKeys
}

func (m *MarketTracker) serialiseMarketTracker() []*snapshot.MarketVolumeTracker {
	markets := make([]string, 0, len(m.marketIDMarketTracker))
	for k := range m.marketIDMarketTracker {
		markets = append(markets, k)
	}
	sort.Strings(markets)

	mt := make([]*snapshot.MarketVolumeTracker, 0, len(markets))
	for _, market := range markets {
		tracker := m.marketIDMarketTracker[market]
		mt = append(mt, &snapshot.MarketVolumeTracker{
			Proposer:     tracker.proposer,
			BonusPaid:    tracker.proposersPaid,
			VolumeTraded: tracker.volumeTraded.String(),
			MarketId:     market,
		})
	}

	return mt
}

func (m *MarketTracker) serialise() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadMarketTracker{
			MarketTracker: m.serialiseMarketTracker(),
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form and hash of the given key.
func (m *MarketTracker) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != marketTrackerkey {
		return nil, nil, ErrSnapshotKeyDoesNotExistForMarketTracker
	}

	if !m.ss.changed {
		return m.ss.serialised, m.ss.hash, nil
	}

	data, err := m.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	m.ss.serialised = data
	m.ss.hash = hash
	m.ss.changed = false
	return data, hash, nil
}

func (m *MarketTracker) GetHash(k string) ([]byte, error) {
	_, hash, err := m.getSerialisedAndHash(k)
	return hash, err
}

func (m *MarketTracker) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := m.getSerialisedAndHash(k)
	return state, nil, err
}

func (m *MarketTracker) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if m.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadMarketTracker:
		return nil, m.restore(ctx, pl.MarketTracker)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (m *MarketTracker) restore(ctx context.Context, data []*snapshot.MarketVolumeTracker) error {
	m.marketIDMarketTracker = make(map[string]*marketTracker, len(data))
	for _, d := range data {
		volume, err := num.UintFromString(d.VolumeTraded, 10)
		if err {
			continue
		}
		m.marketIDMarketTracker[d.MarketId] = &marketTracker{
			proposer:      d.Proposer,
			proposersPaid: d.BonusPaid,
			volumeTraded:  volume,
		}
	}

	m.ss.changed = true
	return nil
}
