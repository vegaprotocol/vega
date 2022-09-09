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
	nodeSelector NodeSelector
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

	if !connectedWallet.Permissions().CanUseKey(params.PublicKey) {
		return nil, requestNotPermittedError(ErrPublicKeyIsNotAllowedToBeUsed)
	}

	txReader := strings.NewReader(params.RawTransaction)
	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(txReader, request); err != nil {
		return nil, invalidParams(fmt.Errorf("could not parse the transaction: %w", err))
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
		return nil, clientRejectionError()
	}

	currentNode, err := h.nodeSelector.Node(ctx)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not find an healthy node: %w", err))
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrNoHealthyNodeAvailable)
	}

	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not get last block from node: %w", err))
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrCouldNotGetLastBlockInformation)
	}

	// Sign the payload.
	inputData, err := wcommands.ToMarshaledInputData(request, lastBlockData.Height, lastBlockData.ChainId)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not marshal input data: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	signature, err := connectedWallet.Wallet.SignTx(params.PublicKey, inputData)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not sign command: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	// Build the transaction.
	tx := commands.NewTransaction(params.PublicKey, inputData, &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	})

	// Generate the proof of work for the transaction.
	txID := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(lastBlockData.Hash, txID, uint(lastBlockData.SpamPowDifficulty), vgcrypto.Sha3)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not compute the proof of work: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}
	tx.Pow = &commandspb.ProofOfWork{
		Tid:   txID,
		Nonce: powNonce,
	}

	h.pipeline.NotifySuccessfulRequest(ctx, traceID)

	return SignTransactionResult{
		Tx: tx,
	}, nil
}

func NewSignTransaction(pipeline Pipeline, nodeSelector NodeSelector, sessions *Sessions) *SignTransaction {
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
