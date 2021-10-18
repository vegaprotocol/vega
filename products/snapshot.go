package products

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type futureSnapshotState struct {
	hash       []byte
	serialised []byte
	changed    bool
}

// serialiseFuture returns the future's data as marshalled bytes.
func (f *Future) serialiseFuture() ([]byte, error) {
	fs := &types.FutureState{
		MarketID:          f.marketID,
		TradingTerminated: f.oracle.data.tradingTerminated,
	}

	if f.oracle.data.settlementPrice != nil {
		fs.SettlementPrice = f.oracle.data.settlementPrice.String()
	}

	pl := types.Payload{
		Data: &types.PayloadFutureState{
			FutureState: fs,
		},
	}

	return proto.Marshal(pl.IntoProto())
}

// get the serialised form and hash of the given key.
func (f *Future) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != f.marketID {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !f.fss.changed {
		return f.fss.serialised, f.fss.hash, nil
	}

	data, err := f.serialiseFuture()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	f.fss.serialised = data
	f.fss.hash = hash
	f.fss.changed = false
	return data, hash, nil
}

func (f *Future) Namespace() types.SnapshotNamespace {
	return types.FutureSnapshot
}

func (f *Future) Keys() []string {
	return []string{f.marketID}
}

func (f *Future) GetHash(k string) ([]byte, error) {
	_, hash, err := f.getSerialisedAndHash(k)
	return hash, err
}

func (f *Future) GetState(k string) ([]byte, error) {
	data, _, err := f.getSerialisedAndHash(k)
	return data, err
}

func (f *Future) Snapshot() (map[string][]byte, error) {
	k := f.marketID
	state, err := f.GetState(k)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{k: state}, nil
}

func (f *Future) LoadState(payload *types.Payload) error {
	if f.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadFutureState:
		return f.restoreFuture(pl.FutureState)
	default:
		return types.ErrUnknownSnapshotType
	}
}

func (f *Future) restoreFuture(fs *types.FutureState) error {
	f.fss.changed = true
	f.oracle.data.tradingTerminated = fs.TradingTerminated

	if len(fs.SettlementPrice) == 0 {
		return nil
	}

	price, overflow := num.UintFromString(fs.SettlementPrice, 10)
	if overflow {
		return errors.New("invalid settlement price, needs to be base 10")
	}
	f.oracle.data.settlementPrice = price
	return nil
}
