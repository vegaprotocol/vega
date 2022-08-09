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
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/mapstructure"
)

type SendTransactionParams struct {
	Token              string `json:"token"`
	PublicKey          string `json:"publicKey"`
	SendingMode        string `json:"sendingMode"`
	EncodedTransaction string `json:"encodedTransaction"`
}

type ParsedSendTransactionParams struct {
	Token          string
	PublicKey      string
	SendingMode    apipb.SubmitTransactionRequest_Type
	RawTransaction string
}

type SendTransactionResult struct {
	ReceivedAt time.Time `json:"receivedAt"`
	SentAt     time.Time `json:"sentAt"`
	TxHash     string    `json:"transactionHash"`
}

type SendTransaction struct {
	pipeline     Pipeline
	nodeSelector NodeSelector
	sessions     *Sessions
}

func (h *SendTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := TraceIDFromContext(ctx)

	params, err := validateSendTransactionParams(rawParams)
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
		return nil, invalidParams(fmt.Errorf("couldn't parse the transaction: %w", err))
	}

	request.PubKey = params.PublicKey
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, invalidParams(errs)
	}

	receivedAt := time.Now()
	approved, err := h.pipeline.RequestTransactionReview(ctx, traceID, connectedWallet.Hostname, connectedWallet.Wallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.pipeline, err); errDetails != nil {
			return nil, errDetails
		}
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the transaction review failed: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
	}
	if !approved {
		return nil, clientRejectionError()
	}

	currentNode, err := h.nodeSelector.Node(ctx)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("couldn't find an healthy node: %w", err))
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrNoHealthyNodeAvailable)
	}

	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("couldn't get last block: %w", err))
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrCouldNotGetLastBlockInformation)
	}

	// Sign the payload.
	inputData, err := wcommands.ToMarshaledInputData(request, lastBlockData.Height)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("couldn't marshal input data: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
	}

	signature, err := connectedWallet.Wallet.SignTx(params.PublicKey, inputData)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("couldn't sign command: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
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
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("couldn't compute the proof of work: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
	}
	tx.Pow = &commandspb.ProofOfWork{
		Tid:   txID,
		Nonce: powNonce,
	}

	sentAt := time.Now()
	txHash, err := currentNode.SendTransaction(ctx, tx, params.SendingMode)
	if err != nil {
		h.notifyTransactionStatus(ctx, traceID, txHash, tx, err, sentAt)
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrTransactionFailed)
	}

	h.notifyTransactionStatus(ctx, traceID, txHash, tx, err, sentAt)

	return SendTransactionResult{
		ReceivedAt: receivedAt,
		SentAt:     sentAt,
		TxHash:     txHash,
	}, nil
}

func (h *SendTransaction) notifyTransactionStatus(ctx context.Context, traceID, txHash string, tx *commandspb.Transaction, err error, sentAt time.Time) {
	m := jsonpb.Marshaler{
		EmitDefaults: true,
		Indent:       "  ",
	}
	humanReadableTx, mErr := m.MarshalToString(tx)
	if mErr != nil {
		// We ignore this error as it's not critical to have the transaction
		// sent back. At least, we can transmit the transaction hash so the
		// client front-end can redirect to the block explorer.
		humanReadableTx = ""
	}
	h.pipeline.NotifyTransactionStatus(ctx, traceID, txHash, humanReadableTx, err, sentAt)
}

func NewSendTransaction(pipeline Pipeline, nodeSelector NodeSelector, sessions *Sessions) *SendTransaction {
	return &SendTransaction{
		pipeline:     pipeline,
		nodeSelector: nodeSelector,
		sessions:     sessions,
	}
}

func validateSendTransactionParams(rawParams jsonrpc.Params) (ParsedSendTransactionParams, error) {
	if rawParams == nil {
		return ParsedSendTransactionParams{}, ErrParamsRequired
	}

	params := SendTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ParsedSendTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return ParsedSendTransactionParams{}, ErrConnectionTokenIsRequired
	}

	if params.PublicKey == "" {
		return ParsedSendTransactionParams{}, ErrPublicKeyIsRequired
	}

	if params.SendingMode == "" {
		return ParsedSendTransactionParams{}, ErrSendingModeIsRequired
	}

	isValidSendingMode := false
	var sendingMode apipb.SubmitTransactionRequest_Type
	for tp, sm := range apipb.SubmitTransactionRequest_Type_value {
		if tp == params.SendingMode {
			isValidSendingMode = true
			sendingMode = apipb.SubmitTransactionRequest_Type(sm)
		}
	}
	if !isValidSendingMode {
		return ParsedSendTransactionParams{}, fmt.Errorf("sending mode %q is not a valid one", params.SendingMode)
	}

	if sendingMode == apipb.SubmitTransactionRequest_TYPE_UNSPECIFIED {
		return ParsedSendTransactionParams{}, ErrSendingModeCannotBeTypeUnspecified
	}

	if params.EncodedTransaction == "" {
		return ParsedSendTransactionParams{}, ErrEncodedTransactionIsRequired
	}

	tx, err := base64.StdEncoding.DecodeString(params.EncodedTransaction)
	if err != nil {
		return ParsedSendTransactionParams{}, ErrEncodedTransactionIsNotValidBase64String
	}

	return ParsedSendTransactionParams{
		Token:          params.Token,
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
		SendingMode:    sendingMode,
	}, nil
}
