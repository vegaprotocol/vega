package node

import (
	"context"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"github.com/cenkalti/backoff"
)

// UponGenesis loads all asset from genesis state.
func (n *NodeCommand) UponGenesis(ctx context.Context, rawstate []byte) (err error) {
	n.Log.Debug("Entering node.NodeCommand.UponGenesis")
	defer func() {
		if err != nil {
			n.Log.Debug("Failure in node.NodeCommand.UponGenesis", logging.Error(err))
		} else {
			n.Log.Debug("Leaving node.NodeCommand.UponGenesis without error")
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
		err := n.loadAsset(ctx, k, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *NodeCommand) loadAsset(ctx context.Context, id string, v *proto.AssetDetails) error {
	aid, err := n.assets.NewAsset(id, types.AssetDetailsFromProto(v))
	if err != nil {
		return fmt.Errorf("error instanciating asset %v", err)
	}

	asset, err := n.assets.Get(aid)
	if err != nil {
		return fmt.Errorf("unable to get asset %v", err)
	}

	if n.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain && asset.IsERC20() {
		return ErrERC20AssetWithNullChain
	}

	// the validation is required only for validators
	if n.conf.IsValidator() {
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

	if err := n.assets.Enable(aid); err != nil {
		n.Log.Error("invalid genesis asset",
			logging.String("asset-details", v.String()),
			logging.Error(err))
		return fmt.Errorf("unable to enable asset: %v", err)
	}

	assetD := asset.Type()
	if err := n.collateral.EnableAsset(ctx, *assetD); err != nil {
		return fmt.Errorf("unable to enable asset in collateral: %v", err)
	}

	n.Log.Info("new asset added successfully",
		logging.String("asset", asset.String()))

	return nil
}
