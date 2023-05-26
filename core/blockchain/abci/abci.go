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

package abci

import (
	"encoding/hex"
	"errors"

	"code.vegaprotocol.io/vega/core/blockchain"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/tendermint/tendermint/abci/types"
)

func (app *App) Info(req types.RequestInfo) types.ResponseInfo {
	if fn := app.OnInfo; fn != nil {
		return fn(req)
	}
	return app.BaseApplication.Info(req)
}

func (app *App) InitChain(req types.RequestInitChain) (resp types.ResponseInitChain) {
	_, err := LoadGenesisState(req.AppStateBytes)
	if err != nil {
		panic(err)
	}

	if fn := app.OnInitChain; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) BeginBlock(req types.RequestBeginBlock) (resp types.ResponseBeginBlock) {
	if fn := app.OnBeginBlock; fn != nil {
		app.ctx, resp = fn(req)
	}
	return
}

func (app *App) EndBlock(req types.RequestEndBlock) (resp types.ResponseEndBlock) {
	if fn := app.OnEndBlock; fn != nil {
		app.ctx, resp = fn(req)
	}
	return
}

func (app *App) Commit() (resp types.ResponseCommit) {
	if fn := app.OnCommit; fn != nil {
		return fn()
	}
	return
}

func (app *App) CheckTx(req types.RequestCheckTx) (resp types.ResponseCheckTx) {
	// first, only decode the transaction but don't validate
	tx, code, err := app.getTx(req.GetTx())
	if err != nil {
		return blockchain.NewResponseCheckTxError(code, err)
	}

	// check for spam and replay
	if fn := app.OnCheckTxSpam; fn != nil {
		resp = fn(tx)
		if resp.IsErr() {
			return AddCommonCheckTxEvents(resp, tx)
		}
	}

	ctx := app.ctx
	if fn := app.OnCheckTx; fn != nil {
		ctx, resp = fn(ctx, req, tx)
		if resp.IsErr() {
			return AddCommonCheckTxEvents(resp, tx)
		}
	}

	// Lookup for check tx, skip if not found
	if fn, ok := app.checkTxs[tx.Command()]; ok {
		if err := fn(ctx, tx); err != nil {
			return AddCommonCheckTxEvents(blockchain.NewResponseCheckTxError(blockchain.AbciTxnInternalError, err), tx)
		}
	}

	// at this point we consider the Tx as valid, so we add it to
	// the cache to be consumed by DeliveryTx
	if resp.IsOK() {
		app.cacheTx(req.Tx, tx)
	}

	return AddCommonCheckTxEvents(resp, tx)
}

func (app *App) DeliverTx(req types.RequestDeliverTx) (resp types.ResponseDeliverTx) {
	// first, only decode the transaction but don't validate
	tx, code, err := app.getTx(req.GetTx())
	if err != nil {
		return blockchain.NewResponseDeliverTxError(code, err)
	}
	app.removeTxFromCache(req.GetTx())

	// check for spam and replay
	if fn := app.OnDeliverTxSpam; fn != nil {
		resp = fn(app.ctx, tx)
		if resp.IsErr() {
			return AddCommonDeliverTxEvents(resp, tx)
		}
	}

	// It's been validated by CheckTx so we can skip the validation here
	ctx := app.ctx
	if fn := app.OnDeliverTx; fn != nil {
		ctx, resp = fn(ctx, req, tx)
		if resp.IsErr() {
			return AddCommonDeliverTxEvents(resp, tx)
		}
	}

	// Lookup for deliver tx, fail if not found
	fn := app.deliverTxs[tx.Command()]
	if fn == nil {
		return AddCommonDeliverTxEvents(
			blockchain.NewResponseDeliverTxError(blockchain.AbciUnknownCommandError, errors.New("invalid vega command")), tx,
		)
	}

	txHash := hex.EncodeToString(tx.Hash())
	ctx = vgcontext.WithTxHash(ctx, txHash)

	if err := fn(ctx, tx); err != nil {
		if perr, ok := err.(MaybePartialError); ok {
			if perr.IsPartial() {
				return AddCommonDeliverTxEvents(
					blockchain.NewResponseDeliverTxError(blockchain.AbciTxnPartialProcessingError, err), tx,
				)
			}
		}

		return AddCommonDeliverTxEvents(
			blockchain.NewResponseDeliverTxError(blockchain.AbciTxnInternalError, err), tx,
		)
	}

	return AddCommonDeliverTxEvents(
		blockchain.NewResponseDeliverTx(types.CodeTypeOK, ""), tx,
	)
}

func (app *App) ListSnapshots(req types.RequestListSnapshots) (resp types.ResponseListSnapshots) {
	if app.OnListSnapshots != nil {
		resp = app.OnListSnapshots(req)
	}
	return
}

func (app *App) OfferSnapshot(req types.RequestOfferSnapshot) (resp types.ResponseOfferSnapshot) {
	if app.OnOfferSnapshot != nil {
		resp = app.OnOfferSnapshot(req)
	}
	return
}

func (app *App) LoadSnapshotChunk(req types.RequestLoadSnapshotChunk) (resp types.ResponseLoadSnapshotChunk) {
	if app.OnLoadSnapshotChunk != nil {
		resp = app.OnLoadSnapshotChunk(req)
	}
	return
}

func (app *App) ApplySnapshotChunk(req types.RequestApplySnapshotChunk) (resp types.ResponseApplySnapshotChunk) {
	if app.OnApplySnapshotChunk != nil {
		resp = app.OnApplySnapshotChunk(app.ctx, req)
	}
	return
}

func AddCommonCheckTxEvents(resp types.ResponseCheckTx, tx Tx) types.ResponseCheckTx {
	resp.Events = getBaseTxEvents(tx)
	return resp
}

func AddCommonDeliverTxEvents(resp types.ResponseDeliverTx, tx Tx) types.ResponseDeliverTx {
	resp.Events = getBaseTxEvents(tx)
	return resp
}

const (
	TxTypeStakeDeposited                     = "StakeDeposited"
	TxTypeStakeRemoved                       = "StakeRemoved"
	TxTypeBuiltinAssetDeposit                = "BuiltinAssetDeposit"
	TxTypeBuiltinAssetWithdrawal             = "BuiltinAssetWithdrawal"
	TxTypeERC20Deposit                       = "ERC20Deposit"
	TxTypeERC20Withdrawal                    = "ERC20Withdrawal"
	TxTypeTransfer                           = "Transfer"
	TxTypeOrderSubmission                    = "OrderSubmission"
	TxTypeOrderCancellation                  = "OrderCancellation"
	TxTypeOrderAmendment                     = "OrderAmendment"
	TxTypeVoteSubmission                     = "VoteSubmission"
	TxTypeWithdrawSubmission                 = "WithdrawSubmission"
	TxTypeLiquidityProvisionSubmission       = "LiquidityProvisionSubmission"
	TxTypeLiquidityProvisionCancellation     = "LiquidityProvisionCancellation"
	TxTypeLiquidityProvisionAmendment        = "LiquidityProvisionAmendment"
	TxTypeSpotLiquidityProvisionSubmission   = "SpotLiquidityProvisionSubmission"
	TxTypeSpotLiquidityProvisionCancellation = "SpotLiquidityProvisionCancellation"
	TxTypeSpotLiquidityProvisionAmendment    = "SpotLiquidityProvisionAmendment"
	TxTypeProposalSubmission                 = "ProposalSubmission"
	TxTypeAnnounceNode                       = "AnnounceNode"
	TxTypeNodeVote                           = "NodeVote"
	TxTypeNodeSignature                      = "NodeSignature"
	TxTypeOracleDataSubmission               = "OracleDataSubmission"
	TxTypeDelegateSubmission                 = "DelegateSubmission"
	TxTypeUndelegateSubmission               = "UndelegateSubmission"
	TxTypeKeyRotateSubmission                = "KeyRotateSubmission"
	TxTypeStateVariableProposal              = "StateVariableProposal"
	TxTypeCancelTransfer                     = "CancelTransfer"
	// TxTypeValidatorHeartbeat                 = "ValidatorHeartbeat"
	TxTypeEthereumKeyRotateSubmission = "EthereumKeyRotateSubmission"
	TxTypeProtocolUpgradeProposal     = "ProtocolUpgradeProposal"
	TxTypeIssueSignatures             = "IssueSignatures"
	TxTypeBatchMarketInstructions     = "BatchMarketInstructions"
	TxTypeERC20MultisigSignerAdded    = "ERC20MultisigSignerAdded"
	TxTypeERC20MultisigSignerRemoved  = "ERC20MultisigSignerRemoved"
)

type TxType struct {
	Type     string
	Sender   string
	Receiver string
}

func GetTxType(tx Tx) (txt TxType) {
	switch c := tx.GetCmd().(type) {
	case *commandspb.ChainEvent:
		if se := c.GetStakingEvent(); se != nil {
			if sed := se.GetStakeDeposited(); sed != nil {
				txt.Type = TxTypeStakeDeposited
				txt.Receiver = sed.GetVegaPublicKey()
			}
			if ser := se.GetStakeRemoved(); ser != nil {
				txt.Type = TxTypeStakeRemoved
				txt.Receiver = ser.GetVegaPublicKey()
			}
		}
		if bi := c.GetBuiltin(); bi != nil {
			if bid := bi.GetDeposit(); bid != nil {
				txt.Type = TxTypeBuiltinAssetDeposit
				txt.Receiver = bid.GetPartyId()
			}
			if biw := bi.GetWithdrawal(); biw != nil {
				txt.Type = TxTypeBuiltinAssetWithdrawal
				txt.Sender = biw.GetPartyId()
			}
		}
		if e20 := c.GetErc20(); e20 != nil {
			if e20d := e20.GetDeposit(); e20d != nil {
				txt.Type = TxTypeERC20Deposit
				txt.Receiver = e20d.GetTargetPartyId()
			}
			if e20w := e20.GetWithdrawal(); e20w != nil {
				txt.Type = TxTypeERC20Withdrawal
				txt.Sender = tx.Party() // TODO: what else to do here?
			}
		}
		if erc20ms := c.GetErc20Multisig(); erc20ms != nil {
			if erc20msadd := erc20ms.GetSignerAdded(); erc20msadd != nil {
				txt.Type = TxTypeERC20MultisigSignerAdded
				txt.Sender = erc20msadd.GetNewSigner() // TODO: what else to do here?
			}
			if erc20msrem := erc20ms.GetSignerRemoved(); erc20msrem != nil {
				txt.Type = TxTypeERC20MultisigSignerRemoved
				txt.Sender = erc20msrem.GetOldSigner() // TODO: what else to do here?
			}
		}
	case *commandspb.Transfer:
		txt.Type = TxTypeTransfer
		txt.Receiver = c.GetTo()
		txt.Sender = tx.Party() // TODO: ???
	case *commandspb.OrderSubmission:
		txt.Type = TxTypeOrderSubmission
		txt.Sender = tx.Party()
	case *commandspb.OrderCancellation:
		txt.Type = TxTypeOrderCancellation
		txt.Sender = tx.Party()
	case *commandspb.OrderAmendment:
		txt.Type = TxTypeOrderAmendment
		txt.Sender = tx.Party()
	case *commandspb.VoteSubmission:
		txt.Type = TxTypeVoteSubmission
		txt.Sender = tx.Party()
	case *commandspb.WithdrawSubmission:
		txt.Type = TxTypeWithdrawSubmission
		txt.Sender = tx.Party()
	case *commandspb.LiquidityProvisionSubmission:
		txt.Type = TxTypeLiquidityProvisionSubmission
		txt.Sender = tx.Party()
	case *commandspb.LiquidityProvisionCancellation:
		txt.Type = TxTypeLiquidityProvisionCancellation
		txt.Sender = tx.Party()
	case *commandspb.LiquidityProvisionAmendment:
		txt.Type = TxTypeLiquidityProvisionAmendment
		txt.Sender = tx.Party()
	case *commandspb.SpotLiquidityProvisionSubmission:
		txt.Type = TxTypeSpotLiquidityProvisionSubmission
		txt.Sender = tx.Party()
	case *commandspb.SpotLiquidityProvisionCancellation:
		txt.Type = TxTypeSpotLiquidityProvisionCancellation
		txt.Sender = tx.Party()
	case *commandspb.SpotLiquidityProvisionAmendment:
		txt.Type = TxTypeSpotLiquidityProvisionAmendment
		txt.Sender = tx.Party()
	case *commandspb.ProposalSubmission:
		txt.Type = TxTypeProposalSubmission
		txt.Sender = tx.Party()
	case *commandspb.AnnounceNode:
		txt.Type = TxTypeAnnounceNode
		txt.Sender = tx.Party()
	case *commandspb.NodeVote: // TODO: do we need this?
		txt.Type = TxTypeNodeVote
		txt.Sender = tx.Party()
	case *commandspb.NodeSignature:
		txt.Type = TxTypeNodeSignature
		txt.Sender = tx.Party()
	case *commandspb.OracleDataSubmission:
		txt.Type = TxTypeOracleDataSubmission
		txt.Sender = tx.Party()
	case *commandspb.DelegateSubmission:
		txt.Type = TxTypeDelegateSubmission
		txt.Sender = tx.Party()
	case *commandspb.UndelegateSubmission:
		txt.Type = TxTypeUndelegateSubmission
		txt.Sender = tx.Party()
	case *commandspb.KeyRotateSubmission:
		txt.Type = TxTypeKeyRotateSubmission
		txt.Sender = tx.Party()
	case *commandspb.StateVariableProposal:
		txt.Type = TxTypeStateVariableProposal
		txt.Sender = tx.Party()
	case *commandspb.CancelTransfer:
		txt.Type = TxTypeCancelTransfer
		txt.Sender = tx.Party()
	// case *commandspb.ValidatorHeartbeat: TODO: do we need this?
	// 	txt.Type = TxTypeValidatorHeartbeat
	case *commandspb.EthereumKeyRotateSubmission:
		txt.Type = TxTypeEthereumKeyRotateSubmission
		txt.Sender = tx.Party()
	case *commandspb.ProtocolUpgradeProposal:
		txt.Type = TxTypeProtocolUpgradeProposal
		txt.Sender = tx.Party()
	case *commandspb.IssueSignatures:
		txt.Type = TxTypeIssueSignatures
		txt.Sender = tx.Party()
	case *commandspb.BatchMarketInstructions:
		txt.Type = TxTypeBatchMarketInstructions
		txt.Sender = tx.Party()
	}
	return
}

func getBaseTxEvents(tx Tx) []types.Event {
	base := []types.Event{
		{
			Type: "tx",
			Attributes: []types.EventAttribute{
				{
					Key:   []byte("submitter"),
					Value: []byte(tx.PubKeyHex()),
					Index: true,
				},
			},
		},
		{
			Type: "command",
			Attributes: []types.EventAttribute{
				{
					Key:   []byte("type"),
					Value: []byte(tx.Command().String()),
					Index: true,
				},
			},
		},
	}

	commandAttributes := []types.EventAttribute{}

	cmd := tx.GetCmd()
	if cmd == nil {
		return base
	}

	txt := GetTxType(tx)

	if txt.Type != "" {
		base[0].Attributes = append(base[0].Attributes, types.EventAttribute{
			Key:   []byte("type"),
			Value: []byte(txt.Type),
			Index: true,
		})
	}

	if txt.Sender != "" {
		base[0].Attributes = append(base[0].Attributes, types.EventAttribute{
			Key:   []byte("sender"),
			Value: []byte(txt.Sender),
			Index: true,
		})
	}

	if txt.Receiver != "" {
		base[0].Attributes = append(base[0].Attributes, types.EventAttribute{
			Key:   []byte("receiver"),
			Value: []byte(txt.Receiver),
			Index: true,
		})
	}

	var market string
	if m, ok := cmd.(interface{ GetMarketId() string }); ok {
		market = m.GetMarketId()
	}
	if m, ok := cmd.(interface{ GetMarket() string }); ok {
		market = m.GetMarket()
	}
	if len(market) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   []byte("market"),
			Value: []byte(market),
			Index: true,
		})
	}

	var asset string
	if m, ok := cmd.(interface{ GetAssetId() string }); ok {
		asset = m.GetAssetId()
	}
	if m, ok := cmd.(interface{ GetAsset() string }); ok {
		asset = m.GetAsset()
	}
	if len(asset) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   []byte("asset"),
			Value: []byte(asset),
			Index: true,
		})
	}

	var reference string
	if m, ok := cmd.(interface{ GetReference() string }); ok {
		reference = m.GetReference()
	}
	if len(reference) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   []byte("reference"),
			Value: []byte(reference),
			Index: true,
		})
	}

	var proposal string
	if m, ok := cmd.(interface{ GetProposalId() string }); ok {
		proposal = m.GetProposalId()
	}
	if len(proposal) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   []byte("proposal"),
			Value: []byte(proposal),
			Index: true,
		})
	}

	if len(commandAttributes) > 0 {
		base[1].Attributes = append(base[1].Attributes, commandAttributes...)
	}

	return base
}

type MaybePartialError interface {
	error
	IsPartial() bool
}
