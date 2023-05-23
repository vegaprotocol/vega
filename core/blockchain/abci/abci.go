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
	TxTypeValidatorHeartbeat                 = "ValidatorHeartbeat"
	TxTypeEthereumKeyRotateSubmission        = "EthereumKeyRotateSubmission"
	TxTypeProtocolUpgradeProposal            = "ProtocolUpgradeProposal"
	TxTypeIssueSignatures                    = "IssueSignatures"
	TxTypeBatchMarketInstructions            = "BatchMarketInstructions"
)

type txType struct {
	Type     string
	Sender   string
	Receiver string
}

func GetTxType(tx Tx) (txt txType) {
	if tx.GetCmd() == nil {
		return
	}

	switch c := tx.GetCmd().(type) {
	case *commandspb.InputData_ChainEvent:
		ce := c.ChainEvent
		if ce == nil {
			return
		}

		if se := ce.GetStakingEvent(); se != nil {
			if sed := se.GetStakeDeposited(); sed != nil {
				txt.Type = TxTypeStakeDeposited
				txt.Receiver = sed.GetVegaPublicKey()
			}
			if ser := se.GetStakeRemoved(); ser != nil {
				txt.Type = TxTypeStakeRemoved
				txt.Receiver = ser.GetVegaPublicKey()
			}
		}
		if bi := ce.GetBuiltin(); bi != nil {
			if bid := bi.GetDeposit(); bid != nil {
				txt.Type = TxTypeBuiltinAssetDeposit
				txt.Receiver = bid.GetPartyId()
			}
			if biw := bi.GetWithdrawal(); biw != nil {
				txt.Type = TxTypeBuiltinAssetWithdrawal
				txt.Sender = biw.GetPartyId()
			}
		}
		if e20 := ce.GetErc20(); e20 != nil {
			if e20d := e20.GetDeposit(); e20d != nil {
				txt.Type = TxTypeERC20Deposit
				txt.Receiver = e20d.GetTargetPartyId()
			}
			if e20w := e20.GetWithdrawal(); e20w != nil {
				txt.Type = TxTypeERC20Withdrawal
				txt.Sender = tx.Party() // TODO: ???
			}
		}
	case *commandspb.InputData_Transfer:
		tr := c.Transfer
		if tr == nil {
			return
		}
		txt.Type = TxTypeTransfer
		txt.Receiver = tr.GetTo()
		txt.Sender = tx.Party() // TODO: ???
	case *commandspb.InputData_OrderSubmission:
		txt.Type = TxTypeOrderSubmission
	case *commandspb.InputData_OrderCancellation:
		txt.Type = TxTypeOrderCancellation
	case *commandspb.InputData_OrderAmendment:
		txt.Type = TxTypeOrderAmendment
	case *commandspb.InputData_VoteSubmission:
		txt.Type = TxTypeVoteSubmission
	case *commandspb.InputData_WithdrawSubmission:
		txt.Type = TxTypeWithdrawSubmission
	case *commandspb.InputData_LiquidityProvisionSubmission:
		txt.Type = TxTypeLiquidityProvisionSubmission
	case *commandspb.InputData_LiquidityProvisionCancellation:
		txt.Type = TxTypeLiquidityProvisionCancellation
	case *commandspb.InputData_LiquidityProvisionAmendment:
		txt.Type = TxTypeLiquidityProvisionAmendment
	case *commandspb.InputData_SpotLiquidityProvisionSubmission:
		txt.Type = TxTypeSpotLiquidityProvisionSubmission
	case *commandspb.InputData_SpotLiquidityProvisionCancellation:
		txt.Type = TxTypeSpotLiquidityProvisionCancellation
	case *commandspb.InputData_SpotLiquidityProvisionAmendment:
		txt.Type = TxTypeSpotLiquidityProvisionAmendment
	case *commandspb.InputData_ProposalSubmission:
		txt.Type = TxTypeProposalSubmission
	case *commandspb.InputData_AnnounceNode:
		txt.Type = TxTypeAnnounceNode
	case *commandspb.InputData_NodeVote:
		txt.Type = TxTypeNodeVote
	case *commandspb.InputData_NodeSignature:
		txt.Type = TxTypeNodeSignature
	case *commandspb.InputData_OracleDataSubmission:
		txt.Type = TxTypeOracleDataSubmission
	case *commandspb.InputData_DelegateSubmission:
		txt.Type = TxTypeDelegateSubmission
	case *commandspb.InputData_UndelegateSubmission:
		txt.Type = TxTypeUndelegateSubmission
	case *commandspb.InputData_KeyRotateSubmission:
		txt.Type = TxTypeKeyRotateSubmission
	case *commandspb.InputData_StateVariableProposal:
		txt.Type = TxTypeStateVariableProposal
	case *commandspb.InputData_CancelTransfer:
		txt.Type = TxTypeCancelTransfer
	case *commandspb.InputData_ValidatorHeartbeat:
		txt.Type = TxTypeValidatorHeartbeat
	case *commandspb.InputData_EthereumKeyRotateSubmission:
		txt.Type = TxTypeEthereumKeyRotateSubmission
	case *commandspb.InputData_ProtocolUpgradeProposal:
		txt.Type = TxTypeProtocolUpgradeProposal
	case *commandspb.InputData_IssueSignatures:
		txt.Type = TxTypeIssueSignatures
	case *commandspb.InputData_BatchMarketInstructions:
		txt.Type = TxTypeBatchMarketInstructions
	}

	return
}

func getBaseTxEvents(tx Tx) []types.Event {
	txt := GetTxType(tx)

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
			Type: "tx",
			Attributes: []types.EventAttribute{
				{
					Key:   []byte("type"),
					Value: []byte(txt.Type),
					Index: true,
				},
			},
		},
		{
			Type: "tx",
			Attributes: []types.EventAttribute{
				{
					Key:   []byte("sender"),
					Value: []byte(txt.Sender),
					Index: true,
				},
			},
		},
		{
			Type: "tx",
			Attributes: []types.EventAttribute{
				{
					Key:   []byte("receiver"),
					Value: []byte(txt.Receiver),
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
