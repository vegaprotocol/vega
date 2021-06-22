package processor

import (
	"context"
	"errors"
	"strings"

	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"
)

var (
	ErrNotAnERC20Event                                = errors.New("not an erc20 event")
	ErrNotABuiltinAssetEvent                          = errors.New("not an builtin asset event")
	ErrUnsupportedEventAction                         = errors.New("unsupported event action")
	ErrChainEventAssetListERC20WithoutEnoughSignature = errors.New("chain event for erc20 asset list received with missing node signatures")
)

func (app *App) processChainEvent(
	ctx context.Context, ce *commandspb.ChainEvent, pubkey []byte, id string,
) error {
	// first verify the event was emitted by a validator
	if !app.top.Exists(pubkey) {
		return ErrChainEventFromNonValidator
	}

	// ack the new event then
	if !app.evtfwd.Ack(ce) {
		// there was an error, or this was already acked
		// but that's not a big issue we just going to ignore that.
		return nil
	}

	// OK the event was newly acknowledged, so now we need to
	// figure out what to do with it.
	switch c := ce.Event.(type) {
	case *commandspb.ChainEvent_Builtin:
		// Convert from protobuf to local domain type
		ceb, err := types.NewChainEventBuiltinFromProto(c)
		if err != nil {
			return err
		}
		return app.processChainEventBuiltinAsset(ctx, ceb, id, ce.Nonce)
	case *commandspb.ChainEvent_Erc20:
		// Convert from protobuf to local domain type
		ceErc, err := types.NewChainEventERC20FromProto(c)
		if err != nil {
			return err
		}
		return app.processChainEventERC20(ctx, ceErc, id, ce.TxId)
	case *commandspb.ChainEvent_Btc:
		return errors.New("BTC Event not implemented")
	case *commandspb.ChainEvent_Validator:
		return errors.New("validator Event not implemented")
	default:
		return ErrUnsupportedChainEvent
	}
}

func (app *App) processChainEventBuiltinAsset(ctx context.Context, ce *types.ChainEvent_Builtin, id string, nonce uint64) error {
	evt := ce.Builtin
	if evt == nil {
		return ErrNotABuiltinAssetEvent
	}

	switch act := evt.Action.(type) {
	case *types.BuiltinAssetEventDeposit:
		if err := app.checkVegaAssetID(act.Deposit, "BuiltinAsset.Deposit"); err != nil {
			return err
		}
		return app.banking.DepositBuiltinAsset(ctx, act.Deposit, id, nonce)
	case *types.BuiltinAssetEventWithdrawal:
		if err := app.checkVegaAssetID(act.Withdrawal, "BuiltinAsset.Withdrawal"); err != nil {
			return err
		}
		return errors.New("unreachable")
	default:
		return ErrUnsupportedEventAction
	}
}

func (app *App) processChainEventERC20(
	ctx context.Context, ce *types.ChainEventERC20, id, txId string,
) error {
	evt := ce.ERC20
	if evt == nil {
		return ErrNotAnERC20Event
	}

	switch act := evt.Action.(type) {
	case *types.ERC20EventAssetList:
		act.AssetList.VegaAssetId = strings.TrimPrefix(act.AssetList.VegaAssetId, "0x")
		if err := app.checkVegaAssetID(act.AssetList, "ERC20.AssetList"); err != nil {
			return err
		}
		// now check that the notary is GO for this asset
		_, ok := app.notary.IsSigned(
			ctx,
			act.AssetList.VegaAssetId,
			commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW)
		if !ok {
			return ErrChainEventAssetListERC20WithoutEnoughSignature
		}
		return app.banking.EnableERC20(ctx, act.AssetList, evt.Block, evt.Index, txId)
	case *types.ERC20EventAssetDelist:
		return errors.New("ERC20.AssetDelist not implemented")
	case *types.ERC20EventDeposit:
		act.Deposit.VegaAssetId = strings.TrimPrefix(act.Deposit.VegaAssetId, "0x")

		if err := app.checkVegaAssetID(act.Deposit, "ERC20.AssetDeposit"); err != nil {
			return err
		}
		return app.banking.DepositERC20(ctx, act.Deposit, id, evt.Block, evt.Index, txId)
	case *types.ERC20EventWithdrawal:
		act.Withdrawal.VegaAssetId = strings.TrimPrefix(act.Withdrawal.VegaAssetId, "0x")
		if err := app.checkVegaAssetID(act.Withdrawal, "ERC20.AssetWithdrawal"); err != nil {
			return err
		}
		return app.banking.WithdrawalERC20(ctx, act.Withdrawal, evt.Block, evt.Index, txId)
	default:
		return ErrUnsupportedEventAction
	}
}

type HasVegaAssetID interface {
	GetVegaAssetId() string
}

func (app *App) checkVegaAssetID(a HasVegaAssetID, action string) error {
	id := a.GetVegaAssetId()
	_, err := app.assets.Get(id)
	if err != nil {
		app.log.Error("invalid vega asset ID",
			logging.String("action", action),
			logging.Error(err),
			logging.String("asset-id", id))
		return err
	}
	return nil
}
