package service

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/datanode/service OrderStore,ChainStore,MarketStore,MarketDataStore,PositionStore
