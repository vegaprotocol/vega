// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package protocol

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/cenkalti/backoff"
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

	for k, v := range state {
		err := svcs.loadAsset(ctx, k, v)
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

	// if svcs.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain && asset.IsERC20() {
	// 	return ErrERC20AssetWithNullChain
	// }

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
