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

package processor

import (
	"context"
	"errors"
	"strings"

	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	vgproto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

var (
	ErrNotAnERC20Event                                = errors.New("not an erc20 event")
	ErrNotABuiltinAssetEvent                          = errors.New("not an builtin asset event")
	ErrUnsupportedEventAction                         = errors.New("unsupported event action")
	ErrChainEventAssetListERC20WithoutEnoughSignature = errors.New("chain event for erc20 asset list received with missing node signatures")
)

func (app *App) processChainEvent(
	ctx context.Context, ce *commandspb.ChainEvent, pubkey string, id string,
) error {
	if app.log.GetLevel() <= logging.DebugLevel {
		app.log.Debug("received chain event",
			logging.String("event", ce.String()),
			logging.String("pubkey", pubkey),
		)
	}

	// first verify the event was emitted by a validator
	if !app.top.IsValidatorVegaPubKey(pubkey) {
		app.log.Debug("received chain event from non-validator",
			logging.String("event", ce.String()),
			logging.String("pubkey", pubkey),
		)
		return ErrChainEventFromNonValidator
	}

	// let the topology know who was the validator that forwarded the event
	app.top.AddForwarder(pubkey)

	// ack the new event then
	if !app.evtfwd.Ack(ce) {
		// there was an error, or this was already acked
		// but that's not a big issue we just going to ignore that.
		return nil
	}

	// OK the event was newly acknowledged, so now we need to
	// figure out what to do with it.
	switch c := ce.Event.(type) {
	case *commandspb.ChainEvent_StakingEvent:
		blockNumber := c.StakingEvent.Block
		logIndex := c.StakingEvent.Index
		switch evt := c.StakingEvent.Action.(type) {
		case *vgproto.StakingEvent_TotalSupply:
			stakeTotalSupply, err := types.StakeTotalSupplyFromProto(evt.TotalSupply)
			if err != nil {
				return err
			}
			return app.stakingAccounts.ProcessStakeTotalSupply(ctx, stakeTotalSupply)
		case *vgproto.StakingEvent_StakeDeposited:
			stakeDeposited, err := types.StakeDepositedFromProto(
				evt.StakeDeposited, blockNumber, logIndex, ce.TxId, id)
			if err != nil {
				return err
			}
			return app.stake.ProcessStakeDeposited(ctx, stakeDeposited)
		case *vgproto.StakingEvent_StakeRemoved:
			stakeRemoved, err := types.StakeRemovedFromProto(
				evt.StakeRemoved, blockNumber, logIndex, ce.TxId, id)
			if err != nil {
				return err
			}
			return app.stake.ProcessStakeRemoved(ctx, stakeRemoved)
		default:
			return errors.New("unsupported StakingEvent")
		}
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
	case *commandspb.ChainEvent_Erc20Multisig:
		blockNumber := c.Erc20Multisig.Block
		logIndex := c.Erc20Multisig.Index
		switch pevt := c.Erc20Multisig.Action.(type) {
		case *vgproto.ERC20MultiSigEvent_SignerAdded:
			evt, err := types.SignerEventFromSignerAddedProto(
				pevt.SignerAdded, blockNumber, logIndex, ce.TxId, id)
			if err != nil {
				return err
			}
			return app.erc20MultiSigTopology.ProcessSignerEvent(evt)
		case *vgproto.ERC20MultiSigEvent_SignerRemoved:
			evt, err := types.SignerEventFromSignerRemovedProto(
				pevt.SignerRemoved, blockNumber, logIndex, ce.TxId, id)
			if err != nil {
				return err
			}
			return app.erc20MultiSigTopology.ProcessSignerEvent(evt)
		case *vgproto.ERC20MultiSigEvent_ThresholdSet:
			evt, err := types.SignerThresholdSetEventFromProto(
				pevt.ThresholdSet, blockNumber, logIndex, ce.TxId, id)
			if err != nil {
				return err
			}
			return app.erc20MultiSigTopology.ProcessThresholdEvent(evt)
		default:
			return errors.New("unsupported erc20 multisig event")
		}
	case *commandspb.ChainEvent_ContractCall:
		callResult, err := ethcall.EthereumContractCallResultFromProto(c.ContractCall)
		if err != nil {
			app.log.Error("received invalid contract call", logging.Error(err), logging.String("call", c.ContractCall.String()))
			return err
		}

		chainID := uint64(1)
		if callResult.L2ChainID != nil {
			chainID = *callResult.L2ChainID
		}

		if chainID == 1 {
			return app.oracles.EthereumOraclesVerifier.ProcessEthereumContractCallResult(callResult)
		}

		// use l2 then
		return app.oracles.EthereumL2OraclesVerifier.ProcessEthereumContractCallResult(callResult)

	default:
		return ErrUnsupportedChainEvent
	}
}

func (app *App) processChainEventBuiltinAsset(ctx context.Context, ce *types.ChainEventBuiltin, id string, nonce uint64) error {
	evt := ce.Builtin // nolint
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
	ctx context.Context, ce *types.ChainEventERC20, id, txID string,
) error {
	evt := ce.ERC20 // nolint
	if evt == nil {
		return ErrNotAnERC20Event
	}

	switch act := evt.Action.(type) {
	case *types.ERC20EventAssetList:
		act.AssetList.VegaAssetID = strings.TrimPrefix(act.AssetList.VegaAssetID, "0x")
		if err := app.checkVegaAssetID(act.AssetList, "ERC20.AssetList"); err != nil {
			return err
		}
		// now check that the notary is GO for this asset
		_, ok := app.notary.IsSigned(
			ctx,
			act.AssetList.VegaAssetID,
			commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW)
		if !ok {
			return ErrChainEventAssetListERC20WithoutEnoughSignature
		}
		return app.banking.EnableERC20(ctx, act.AssetList, id, evt.Block, evt.Index, txID)
	case *types.ERC20EventAssetDelist:
		return errors.New("ERC20.AssetDelist not implemented")
	case *types.ERC20EventDeposit:
		act.Deposit.VegaAssetID = strings.TrimPrefix(act.Deposit.VegaAssetID, "0x")
		if err := app.checkVegaAssetID(act.Deposit, "ERC20.AssetDeposit"); err != nil {
			return err
		}
		return app.banking.DepositERC20(ctx, act.Deposit, id, evt.Block, evt.Index, txID)
	case *types.ERC20EventWithdrawal:
		act.Withdrawal.VegaAssetID = strings.TrimPrefix(act.Withdrawal.VegaAssetID, "0x")
		if err := app.checkVegaAssetID(act.Withdrawal, "ERC20.AssetWithdrawal"); err != nil {
			return err
		}
		return app.banking.ERC20WithdrawalEvent(ctx, act.Withdrawal, evt.Block, evt.Index, txID)
	case *types.ERC20EventAssetLimitsUpdated:
		act.AssetLimitsUpdated.VegaAssetID = strings.TrimPrefix(act.AssetLimitsUpdated.VegaAssetID, "0x")
		if err := app.checkVegaAssetID(act.AssetLimitsUpdated, "ERC20.AssetLimitsUpdated"); err != nil {
			return err
		}
		return app.banking.UpdateERC20(ctx, act.AssetLimitsUpdated, id, evt.Block, evt.Index, txID)
	case *types.ERC20EventBridgeStopped:
		return app.banking.BridgeStopped(
			ctx, act.BridgeStopped, id, evt.Block, evt.Index, txID)
	case *types.ERC20EventBridgeResumed:
		return app.banking.BridgeResumed(
			ctx, act.BridgeResumed, id, evt.Block, evt.Index, txID)
	default:
		return ErrUnsupportedEventAction
	}
}

type HasVegaAssetID interface {
	GetVegaAssetID() string
}

func (app *App) checkVegaAssetID(a HasVegaAssetID, action string) error {
	id := a.GetVegaAssetID()
	if _, err := app.assets.Get(id); err != nil {
		app.log.Error("invalid vega asset ID",
			logging.String("action", action),
			logging.Error(err),
			logging.String("asset-id", id))
		return err
	}
	return nil
}
