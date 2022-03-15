package protocol

import (
	"context"
	"errors"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
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
	aid, err := svcs.assets.NewAsset(id, types.AssetDetailsFromProto(v))
	if err != nil {
		return fmt.Errorf("error instanciating asset %v", err)
	}

	asset, err := svcs.assets.Get(aid)
	if err != nil {
		return fmt.Errorf("unable to get asset %v", err)
	}

	if svcs.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain && asset.IsERC20() {
		return ErrERC20AssetWithNullChain
	}

	// the validation is required only for validators
	if svcs.conf.IsValidator() {
		// just a simple backoff here
		err = backoff.Retry(
			func() error {
				err := asset.Validate()
				if !asset.IsValid() {
					return err
				}
				return nil
			},
			backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5),
		)
		if err != nil {
			return fmt.Errorf("unable to instantiate asset \"%s\": %w", v.Name, err)
		}
	} else {
		asset.SetValidNonValidator()
	}

	if err := svcs.assets.Enable(aid); err != nil {
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
