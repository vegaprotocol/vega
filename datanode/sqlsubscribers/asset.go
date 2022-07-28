// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"
	"fmt"
	"math"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
	"github.com/pkg/errors"

	"github.com/shopspring/decimal"
)

type AssetEvent interface {
	events.Event
	Asset() vega.Asset
}

type AssetStore interface {
	Add(context.Context, entities.Asset) error
}

type Asset struct {
	subscriber
	store AssetStore
	log   *logging.Logger
}

func NewAsset(store AssetStore, log *logging.Logger) *Asset {
	return &Asset{
		store: store,
		log:   log,
	}
}

func (a *Asset) Types() []events.Type {
	return []events.Type{events.AssetEvent}
}

func (as *Asset) Push(ctx context.Context, evt events.Event) error {
	return as.consume(ctx, evt.(AssetEvent))
}

func (as *Asset) consume(ctx context.Context, ae AssetEvent) error {
	err := as.addAsset(ctx, ae.Asset(), as.vegaTime)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (as *Asset) addAsset(ctx context.Context, va vega.Asset, vegaTime time.Time) error {
	totalSupply, err := decimal.NewFromString(va.Details.TotalSupply)
	if err != nil {
		return errors.Errorf("bad total supply '%v'", va.Details.TotalSupply)
	}

	quantum, err := decimal.NewFromString(va.Details.Quantum)
	if err != nil {
		return errors.Errorf("bad quantum '%v'", va.Details.Quantum)
	}

	var source, erc20Contract string
	lifetimeLimit := decimal.Zero
	withdrawalThreshold := decimal.Zero
	switch src := va.Details.Source.(type) {
	case *vega.AssetDetails_BuiltinAsset:
		source = src.BuiltinAsset.MaxFaucetAmountMint
	case *vega.AssetDetails_Erc20:
		erc20Contract = src.Erc20.ContractAddress
		if src.Erc20.LifetimeLimit != "" {
			res, err := decimal.NewFromString(src.Erc20.LifetimeLimit)
			if err != nil {
				return fmt.Errorf("couldn't parse lifetime_limit: %w", err)
			}
			lifetimeLimit = res
		}
		if src.Erc20.WithdrawThreshold != "" {
			res, err := decimal.NewFromString(src.Erc20.WithdrawThreshold)
			if err != nil {
				return fmt.Errorf("couldn't parse withdraw_threshold: %w", err)
			}
			withdrawalThreshold = res
		}
	default:
		return errors.Errorf("unknown asset source: %v", source)
	}

	if va.Details.Decimals > math.MaxInt {
		return errors.Errorf("decimals value will cause integer overflow: %d", va.Details.Decimals)
	}

	decimals := int(va.Details.Decimals)

	asset := entities.Asset{
		ID:                entities.NewAssetID(va.Id),
		Name:              va.Details.Name,
		Symbol:            va.Details.Symbol,
		TotalSupply:       totalSupply,
		Decimals:          decimals,
		Quantum:           quantum,
		Source:            source,
		ERC20Contract:     erc20Contract,
		VegaTime:          vegaTime,
		LifetimeLimit:     lifetimeLimit,
		WithdrawThreshold: withdrawalThreshold,
		Status:            entities.AssetStatus(va.Status),
	}

	return errors.WithStack(as.store.Add(ctx, asset))
}
