package sqlsubscribers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"

	"github.com/shopspring/decimal"
)

type AssetEvent interface {
	events.Event
	Asset() types.Asset
}

type AssetStore interface {
	Add(entities.Asset) error
}

type Asset struct {
	*subscribers.Base
	store      AssetStore
	log        *logging.Logger
	blockStore BlockStore
}

func NewAsset(ctx context.Context, store AssetStore, blockStore BlockStore, log *logging.Logger) *Asset {
	return &Asset{
		Base:       subscribers.NewBase(ctx, 0, true),
		store:      store,
		blockStore: blockStore,
		log:        log,
	}
}

func (as *Asset) Types() []events.Type {
	return []events.Type{
		events.AssetEvent,
	}
}

func (as *Asset) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(AssetEvent); ok {
			as.consume(ae)
		}
	}
}

func (as *Asset) consume(ae AssetEvent) {
	as.log.Debug("AssetEvent: ",
		logging.Int64("block", ae.BlockNr()),
		logging.String("assetId", ae.Asset().Id))

	block, err := as.blockStore.WaitForBlockHeight(ae.BlockNr())
	if err != nil {
		as.log.Error("can't add asset because we don't have block")
		return
	}

	err = as.addAsset(ae.Asset(), block.VegaTime)
	if err != nil {
		as.log.Error("adding asset", logging.Error(err))
	}
}

func (as *Asset) addAsset(va types.Asset, vegaTime time.Time) error {
	id := entities.MakeAssetID(va.Id)

	totalSupply, err := decimal.NewFromString(va.Details.TotalSupply)
	if err != nil {
		return fmt.Errorf("bad total supply '%v'", va.Details.TotalSupply)
	}

	quantum, err := strconv.Atoi(va.Details.Quantum)
	if err != nil {
		return fmt.Errorf("bad quantum '%v'", va.Details.Quantum)
	}

	asset := entities.Asset{
		ID:          id,
		Name:        va.Details.Name,
		Symbol:      va.Details.Symbol,
		TotalSupply: totalSupply,
		Quantum:     quantum,
		VegaTime:    vegaTime,
	}

	err = as.store.Add(asset)
	if err != nil {
		return fmt.Errorf("adding asset to store: %w", err)
	}
	return nil
}
