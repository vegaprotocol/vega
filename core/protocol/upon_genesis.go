// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package protocol

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/cenkalti/backoff"
	"golang.org/x/exp/maps"
)

var (
	ErrUnknownChainProvider    = errors.New("unknown chain provider")
	ErrERC20AssetWithNullChain = errors.New("cannot use ERC20 asset with nullchain")
)

// UponGenesis loads all asset from genesis state.
func (svcs *allServices) UponGenesis(ctx context.Context, rawstate []byte) (err error) {
	svcs.log.Debug("Entering node.NodeCommand.UponGenesis")
	defer func() {
		if err != nil {
			svcs.log.Debug("Failure in node.NodeCommand.UponGenesis", logging.Error(err))
		} else {
			svcs.log.Debug("Leaving node.NodeCommand.UponGenesis without error")
		}
	}()

	state, err := assets.LoadGenesisState(rawstate)
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}

	keys := maps.Keys(state)
	sort.Strings(keys)
	for _, k := range keys {
		err := svcs.loadAsset(ctx, k, state[k])
		if err != nil {
			return err
		}
	}

	return nil
}

func (svcs *allServices) loadAsset(
	ctx context.Context, id string, v *proto.AssetDetails,
) error {
	rawAsset, err := types.AssetDetailsFromProto(v)
	if err != nil {
		return err
	}
	aid, err := svcs.assets.NewAsset(ctx, id, rawAsset)
	if err != nil {
		return fmt.Errorf("error instanciating asset %v", err)
	}

	asset, err := svcs.assets.Get(aid)
	if err != nil {
		return fmt.Errorf("unable to get asset %v", err)
	}

	// the validation is required only for validators
	if svcs.conf.IsValidator() {
		// just a simple backoff here
		err = backoff.Retry(
			func() error {
				return svcs.assets.ValidateAsset(aid)
			},
			backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5),
		)
		if err != nil {
			return fmt.Errorf("unable to instantiate asset \"%s\": %w", v.Name, err)
		}
	} else {
		svcs.assets.ValidateAssetNonValidator(aid)
	}

	if err := svcs.assets.Enable(ctx, aid); err != nil {
		svcs.log.Error("invalid genesis asset",
			logging.String("asset-details", v.String()),
			logging.Error(err))
		return fmt.Errorf("unable to enable asset: %v", err)
	}

	assetD := asset.Type()
	if err := svcs.collateral.EnableAsset(ctx, *assetD); err != nil {
		return fmt.Errorf("unable to enable asset in collateral: %v", err)
	}

	svcs.log.Info("new asset added successfully",
		logging.String("asset", asset.String()))

	return nil
}
