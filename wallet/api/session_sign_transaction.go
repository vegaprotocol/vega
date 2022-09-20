package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/commands"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/api/node"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/mapstructure"
)

type SignTransactionParams struct {
	Token              string `json:"token"`
	PublicKey          string `json:"publicKey"`
	EncodedTransaction string `json:"encodedTransaction"`
}

type ParsedSignTransactionParams struct {
	Token          string
	PublicKey      string
	RawTransaction string
}

type SignTransactionResult struct {
	Tx *commandspb.Transaction `json:"transaction"`
}

type SignTransaction struct {
	pipeline     Pipeline
	nodeSelector node.Selector
	sessions     *Sessions
}

func (h *SignTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := TraceIDFromContext(ctx)

	params, err := validateSignTransactionParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	connectedWallet, err := h.sessions.GetConnectedWallet(params.Token)
	if err != nil {
		return nil, invalidParams(err)
	}

	if !connectedWallet.CanUseKey(params.PublicKey) {
		return nil, requestNotPermittedError(ErrPublicKeyIsNotAllowedToBeUsed)
	}

	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(strings.NewReader(params.RawTransaction), request); err != nil {
		return nil, invalidParams(ErrTransactionIsMalformed)
	}

	request.PubKey = params.PublicKey
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, invalidParams(errs)
	}

	receivedAt := time.Now()
	approved, err := h.pipeline.RequestTransactionSigningReview(ctx, traceID, connectedWallet.Hostname, connectedWallet.Wallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.pipeline, err); errDetails != nil {
			return nil, errDetails
		}
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the transaction review failed: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}
	if !approved {
		return nil, userRejectionError()
	}

	h.pipeline.Log(ctx, traceID, InfoLog, "Looking for a healthy node...")
	currentNode, err := h.nodeSelector.Node(ctx, func(reportType node.ReportType, msg string) {
		h.pipeline.Log(ctx, traceID, LogType(reportType), msg)
	})
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not find a healthy node: %w", err))
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrNoHealthyNodeAvailable)
	}
	h.pipeline.Log(ctx, traceID, SuccessLog, "A healthy node has been found.")

	h.pipeline.Log(ctx, traceID, InfoLog, "Retrieving latest block information...")
	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not get the latest block from the node: %w", err))
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrCouldNotGetLastBlockInformation)
	}
	h.pipeline.Log(ctx, traceID, SuccessLog, "Latest block information has been retrieved.")

	if lastBlockData.ChainId == "" {
		h.pipeline.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not get chainID from node: %w", err))
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrCouldNotGetChainIDFromNode)
	}

	// Sign the payload.
	inputData, err := wcommands.ToMarshaledInputData(request, lastBlockData.Height, lastBlockData.ChainId)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not marshal input data: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	h.pipeline.Log(ctx, traceID, InfoLog, "Signing the transaction...")
	signature, err := connectedWallet.Wallet.SignTx(params.PublicKey, inputData)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not sign command: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}
	h.pipeline.Log(ctx, traceID, SuccessLog, "The transaction has been signed.")

	// Build the transaction.
	tx := commands.NewTransaction(params.PublicKey, inputData, &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	})

	// Generate the proof of work for the transaction.
	h.pipeline.Log(ctx, traceID, InfoLog, "Computing proof-of-work...")
	txID := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(lastBlockData.Hash, txID, uint(lastBlockData.SpamPowDifficulty), lastBlockData.SpamPowHashFunction)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not compute the proof-of-work: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}
	tx.Pow = &commandspb.ProofOfWork{
		Tid:   txID,
		Nonce: powNonce,
	}
	h.pipeline.Log(ctx, traceID, SuccessLog, "The proof-of-work has been computed.")

	h.pipeline.NotifySuccessfulRequest(ctx, traceID)

	return SignTransactionResult{
		Tx: tx,
	}, nil
}

func NewSignTransaction(pipeline Pipeline, nodeSelector node.Selector, sessions *Sessions) *SignTransaction {
	return &SignTransaction{
		pipeline:     pipeline,
		nodeSelector: nodeSelector,
		sessions:     sessions,
	}
}

func validateSignTransactionParams(rawParams jsonrpc.Params) (ParsedSignTransactionParams, error) {
	if rawParams == nil {
		return ParsedSignTransactionParams{}, ErrParamsRequired
	}

	params := SignTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ParsedSignTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return ParsedSignTransactionParams{}, ErrConnectionTokenIsRequired
	}

	if params.PublicKey == "" {
		return ParsedSignTransactionParams{}, ErrPublicKeyIsRequired
	}

	if params.EncodedTransaction == "" {
		return ParsedSignTransactionParams{}, ErrEncodedTransactionIsRequired
	}

	tx, err := base64.StdEncoding.DecodeString(params.EncodedTransaction)
	if err != nil {
		return ParsedSignTransactionParams{}, ErrEncodedTransactionIsNotValidBase64String
	}

	return ParsedSignTransactionParams{
		Token:          params.Token,
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
	}, nil
}
