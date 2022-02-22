package entities

import proto "code.vegaprotocol.io/protos/vega"

type Side = proto.Side

const (
	// Default value, always invalid.
	SideUnspecified Side = proto.Side_SIDE_UNSPECIFIED
	// Buy order.
	SideBuy Side = proto.Side_SIDE_BUY
	// Sell order.
	SideSell Side = proto.Side_SIDE_SELL
)

type TradeType = proto.Trade_Type

const (
	// Default value, always invalid.
	TradeTypeUnspecified TradeType = proto.Trade_TYPE_UNSPECIFIED
	// Normal trading between two parties.
	TradeTypeDefault TradeType = proto.Trade_TYPE_DEFAULT
	// Trading initiated by the network with another party on the book,
	// which helps to zero-out the positions of one or more distressed parties.
	TradeTypeNetworkCloseOutGood TradeType = proto.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD
	// Trading initiated by the network with another party off the book,
	// with a distressed party in order to zero-out the position of the party.
	TradeTypeNetworkCloseOutBad TradeType = proto.Trade_TYPE_NETWORK_CLOSE_OUT_BAD
)
