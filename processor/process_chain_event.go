package processor

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrNotAnERC20Event                                = errors.New("not an erc20 event")
	ErrNotABuiltinAssetEvent                          = errors.New("not an builtin asset event")
	ErrUnsupportedEventAction                         = errors.New("unsupported event action")
	ErrChainEventAssetListERC20WithoutEnoughSignature = errors.New("chain event for erc20 asset list received with missing node signatures")
)

func (p *Processor) processChainEvent(ctx context.Context, ce *types.ChainEvent, pubkey []byte) error {
	// first verify the event was emited by a validator
	if !p.top.Exists(pubkey) {
		return ErrChainEventFromNonValidator
	}

	// ack the new event then
	if !p.evtfwd.Ack(ce) {
		// there was an error, or this was already acked
		// but that's not a big issue we just going to ignore that.
		return nil
	}

	// OK the event was newly acknowledged, so now we need to
	// figure out what to do with it.
	switch ce.Event.(type) {
	case *types.ChainEvent_Builtin:
		return p.processChainEventBuiltinAsset(ctx, ce)
	case *types.ChainEvent_Erc20:
		return p.processChainEventERC20(ctx, ce)
	case *types.ChainEvent_Btc:
		return errors.New("BTC Event not implemented")
	case *types.ChainEvent_Validator:
		return errors.New("Validator Event not implemented")
	default:
		return ErrUnsupportedChainEvent
	}
}

func (p *Processor) processChainEventBuiltinAsset(ctx context.Context, ce *types.ChainEvent) error {
	evt := ce.GetBuiltin()
	if evt == nil {
		return ErrNotABuiltinAssetEvent
	}

	switch act := evt.Action.(type) {
	case *types.BuiltinAssetEvent_Deposit:
		if err := p.checkVegaAssetID(act.Deposit, "BuiltinAsset.Deposit"); err != nil {
			return err
		}
		return p.banking.DepositBuiltinAsset(act.Deposit, ce.Nonce)
	case *types.BuiltinAssetEvent_Withdrawal:
		if err := p.checkVegaAssetID(act.Withdrawal, "BuiltinAsset.Withdrawal"); err != nil {
			return err
		}
		return p.col.Withdraw(ctx, act.Withdrawal.PartyID, act.Withdrawal.VegaAssetID, act.Withdrawal.Amount)
	default:
		return ErrUnsupportedEventAction
	}
}

func (p *Processor) processChainEventERC20(ctx context.Context, ce *types.ChainEvent) error {
	evt := ce.GetErc20()
	if evt == nil {
		return ErrNotAnERC20Event
	}

	switch act := evt.Action.(type) {
	case *types.ERC20Event_AssetList:
		if err := p.checkVegaAssetID(act.AssetList, "ERC20.AssetList"); err != nil {
			return err
		}
		// now check that the notary is GO for this asset
		_, ok := p.notary.IsSigned(
			act.AssetList.VegaAssetID,
			types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW)
		if !ok {
			return ErrChainEventAssetListERC20WithoutEnoughSignature
		}
		return p.banking.EnableERC20(ctx, act.AssetList, evt.Block, evt.Index)
	case *types.ERC20Event_AssetDelist:
		return errors.New("ERC20.AssetDelist not implemented")
	case *types.ERC20Event_Deposit:
		act.Deposit.VegaAssetID = act.Deposit.VegaAssetID[2:]

		if err := p.checkVegaAssetID(act.Deposit, "ERC20.AssetDeposit"); err != nil {
			return err
		}
		return p.banking.DepositERC20(act.Deposit, evt.Block, evt.Index)
	case *types.ERC20Event_Withdrawal:
		act.Withdrawal.VegaAssetID = act.Withdrawal.VegaAssetID[2:]
		if err := p.checkVegaAssetID(act.Withdrawal, "ERC20.AssetWithdrawal"); err != nil {
			return err
		}
		return p.banking.WithdrawalERC20(act.Withdrawal, evt.Block, evt.Index)
	default:
		return ErrUnsupportedEventAction
	}
}

type HasVegaAssetID interface {
	GetVegaAssetID() string
}

func (p *Processor) checkVegaAssetID(a HasVegaAssetID, action string) error {
	id := a.GetVegaAssetID()
	_, err := p.assets.Get(id)
	if err != nil {
		p.log.Error("invalid vega asset ID",
			logging.String("action", action),
			logging.Error(err),
			logging.String("asset-id", id))
		return err
	}
	return nil
}
