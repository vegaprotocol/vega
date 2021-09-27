package matching

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
)

const (
	MARKET_ID          = "market id"
	BUY_BOOK           = "buy book"
	SELL_BOOK          = "sell book"
	LAST_TRADE_PRICE   = "last trade price"
	LAST_TIMESTAMP     = "last timestamp"
	AUCTION            = "auction"
	INDICATIVE_P_AND_V = "indicative price and volume"
)

var keys = []string{MARKET_ID,
	BUY_BOOK,
	SELL_BOOK,
	LAST_TRADE_PRICE,
	LAST_TIMESTAMP,
	AUCTION,
	INDICATIVE_P_AND_V}

func (ob *OrderBook) Keys() []string {
	return keys
}

func (ob *OrderBook) Snapshot() (map[string][]byte, error) {
	state := *new(map[string][]byte)
	for _, k := range keys {
		v, err := ob.GetState(k)
		if err != nil {
			return nil, err
		}
		state[k] = v
	}
	return state, nil
}

func (ob OrderBook) Namespace() types.SnapshotNamespace {
	payload := types.PayloadMatchingBook{}
	return payload.Namespace()
}

func (ob *OrderBook) GetHash(key string) ([]byte, error) {
	b, e := ob.GetState(key)
	if e != nil {
		return nil, e
	}
	h := crypto.Hash(b)
	return h, nil
}

func (ob *OrderBook) GetState(key string) ([]byte, error) {
	switch key {
	case MARKET_ID:
		return ob.getMarketIDState()
	case BUY_BOOK:
		return ob.getBuyBookState()
	case SELL_BOOK:
		return ob.getSellBookState()
	case LAST_TRADE_PRICE:
		return ob.getLastTradePriceState()
	case LAST_TIMESTAMP:
		return ob.getLastTimestampState()
	case INDICATIVE_P_AND_V:
		return ob.getIndicativePAndVState()
	}
	return nil, fmt.Errorf("Unknown key: %s", key)
}

func (ob *OrderBook) getMarketIDState() ([]byte, error) {
	return nil, nil
}

func (ob *OrderBook) getBuyBookState() ([]byte, error) {
	return nil, nil
}

func (ob *OrderBook) getSellBookState() ([]byte, error) {
	return nil, nil
}

func (ob *OrderBook) getLastTradePriceState() ([]byte, error) {
	return nil, nil
}

func (ob *OrderBook) getLastTimestampState() ([]byte, error) {
	return nil, nil
}

func (ob *OrderBook) getIndicativePAndVState() ([]byte, error) {
	return nil, nil
}

/*func (ob *OrderBook) hashState() ([]byte, error) {
	// apparently the payload types can't me marshalled by themselves
	pl := types.Payload{
		Data: s.t,
	}
	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return nil, err
	}
	s.data = data
	s.hash = crypto.Hash(data)
	s.updated = false
	return s.hash, nil
}*/
