package sqlsubscribers

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"

	"github.com/shopspring/decimal"
)

type AssetEvent interface {
	events.Event
	Asset() vega.Asset
}

type AssetStore interface {
	Add(entities.Asset) error
}

type Asset struct {
	store    AssetStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewAsset(store AssetStore, log *logging.Logger) *Asset {
	return &Asset{
		store: store,
		log:   log,
	}
}

func (a *Asset) Type() events.Type {
	return events.AssetEvent
}

func (as *Asset) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		as.vegaTime = e.Time()
	case AssetEvent:
		as.consume(e)
	default:
		as.log.Panic("Unknown event type in transfer response subscriber",
			logging.String("Type", e.Type().String()))
	}
}

func (as *Asset) consume(ae AssetEvent) {
	err := as.addAsset(ae.Asset(), as.vegaTime)
	if err != nil {
		as.log.Error("adding asset", logging.Error(err))
	}
}

func (as *Asset) addAsset(va vega.Asset, vegaTime time.Time) error {
	totalSupply, err := decimal.NewFromString(va.Details.TotalSupply)
	if err != nil {
		return fmt.Errorf("bad total supply '%v'", va.Details.TotalSupply)
	}

	quantum, err := strconv.Atoi(va.Details.Quantum)
	if err != nil {
		return fmt.Errorf("bad quantum '%v'", va.Details.Quantum)
	}

	var source, erc20Contract string

	switch src := va.Details.Source.(type) {
	case *vega.AssetDetails_BuiltinAsset:
		source = src.BuiltinAsset.MaxFaucetAmountMint
	case *vega.AssetDetails_Erc20:
		erc20Contract = src.Erc20.ContractAddress
	default:
		return fmt.Errorf("unknown asset source: %v", source)
	}

	if va.Details.Decimals > math.MaxInt {
		return fmt.Errorf("decimals value will cause integer overflow: %d", va.Details.Decimals)
	}

	decimals := int(va.Details.Decimals)

	asset := entities.Asset{
		ID:            entities.NewAssetID(va.Id),
		Name:          va.Details.Name,
		Symbol:        va.Details.Symbol,
		TotalSupply:   totalSupply,
		Decimals:      decimals,
		Quantum:       quantum,
		Source:        source,
		ERC20Contract: erc20Contract,
		VegaTime:      vegaTime,
	}

	err = as.store.Add(asset)
	if err != nil {
		return fmt.Errorf("adding asset to store: %w", err)
	}
	return nil
}
