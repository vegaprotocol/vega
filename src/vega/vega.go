package vega

type Side uint8

const (
	Buy Side = iota
	Sell
)

type OrderType int8

const (
	GTC OrderType = iota
	GTT
	ENE
	FOK
)


type Order interface {
	GetMarket() string
	GetParty() string
	GetSide() Side
	GetPrice() uint64
	GetSize() uint64
	GetRemainingSize() uint64
	GetType() OrderType
	GetTimestamp()

}