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
	"code.vegaprotocol.io/vega/wallet/api/node"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/mitchellh/mapstructure"
)

type ClientSendTransactionParams struct {
	Token              string `json:"token"`
	PublicKey          string `json:"publicKey"`
	SendingMode        string `json:"sendingMode"`
	EncodedTransaction string `json:"encodedTransaction"`
}

type ClientParsedSendTransactionParams struct {
	Token          string
	PublicKey      string
	SendingMode    apipb.SubmitTransactionRequest_Type
	RawTransaction string
}

type ClientSendTransactionResult struct {
	ReceivedAt time.Time               `json:"receivedAt"`
	SentAt     time.Time               `json:"sentAt"`
	TxHash     string                  `json:"transactionHash"`
	Tx         *commandspb.Transaction `json:"transaction"`
}

type ClientSendTransaction struct {
	interactor   Interactor
	nodeSelector node.Selector
	sessions     *Sessions
}

func (h *ClientSendTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := TraceIDFromContext(ctx)

	params, err := validateSendTransactionParams(rawParams)
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

	txReader := strings.NewReader(params.RawTransaction)
	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(txReader, request); err != nil {
		return nil, invalidParams(ErrTransactionIsMalformed)
	}

	request.PubKey = params.PublicKey
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, invalidParams(errs)
	}

	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID); err != nil {
		return nil, internalError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	receivedAt := time.Now()
	approved, err := h.interactor.RequestTransactionReviewForSending(ctx, traceID, connectedWallet.Hostname, connectedWallet.Wallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
			return nil, errDetails
		}
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the transaction review failed: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
	}
	if !approved {
		return nil, userRejectionError()
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Looking for a healthy node...")
	currentNode, err := h.nodeSelector.Node(ctx, func(reportType node.ReportType, msg string) {
		h.interactor.Log(ctx, traceID, LogType(reportType), msg)
	})
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not find a healthy node: %w", err))
		return nil, networkError(ErrNoHealthyNodeAvailable)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Retrieving latest block information...")
	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not get the latest block from node: %w", err))
		return nil, networkError(ErrCouldNotGetLastBlockInformation)
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "Latest block information has been retrieved.")

	if lastBlockData.ChainID == "" {
		h.interactor.NotifyError(ctx, traceID, NetworkError, ErrCouldNotGetChainIDFromNode)
		return nil, networkError(ErrCouldNotGetChainIDFromNode)
	}

	// Sign the payload.
	rawInputData := wcommands.ToInputData(request, lastBlockData.BlockHeight)
	inputData, err := commands.MarshalInputData(rawInputData)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not marshal input data: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Signing the transaction...")
	signature, err := connectedWallet.Wallet.SignTx(params.PublicKey, commands.BundleInputDataForSigning(inputData, lastBlockData.ChainID))
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not sign command: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "The transaction has been signed.")

	// Build the transaction.
	tx := commands.NewTransaction(params.PublicKey, inputData, &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	})

	// Generate the proof of work for the transaction.
	h.interactor.Log(ctx, traceID, InfoLog, "Computing proof-of-work...")
	txID := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(lastBlockData.BlockHash, txID, uint(lastBlockData.ProofOfWorkDifficulty), vgcrypto.Sha3)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not compute the proof-of-work: %w", err))
		return nil, internalError(ErrCouldNotSendTransaction)
	}
	tx.Pow = &commandspb.ProofOfWork{
		Tid:   txID,
		Nonce: powNonce,
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "The proof-of-work has been computed.")

	sentAt := time.Now()
	h.interactor.Log(ctx, traceID, InfoLog, "Sending the transaction...")
	txHash, err := currentNode.SendTransaction(ctx, tx, params.SendingMode)
	if err != nil {
		h.interactor.NotifyFailedTransaction(ctx, traceID, protoToJSON(rawInputData), protoToJSON(tx), err, sentAt)
		return nil, networkError(ErrTransactionFailed)
	}

	h.interactor.NotifySuccessfulTransaction(ctx, traceID, txHash, protoToJSON(rawInputData), protoToJSON(tx), sentAt)

	return ClientSendTransactionResult{
		ReceivedAt: receivedAt,
		SentAt:     sentAt,
		TxHash:     txHash,
		Tx:         tx,
	}, nil
}

func protoToJSON(tx proto.Message) string {
	m := jsonpb.Marshaler{
		EmitDefaults: true,
		Indent:       "  ",
	}
	jsonProto, mErr := m.MarshalToString(tx)
	if mErr != nil {
		// We ignore this error as it's not critical. At least, we can transmit
		// the transaction hash so the client front-end can redirect to the
		// block explorer.
		jsonProto = ""
	}
	return jsonProto
}

func NewSendTransaction(interactor Interactor, nodeSelector node.Selector, sessions *Sessions) *ClientSendTransaction {
	return &ClientSendTransaction{
		interactor:   interactor,
		nodeSelector: nodeSelector,
		sessions:     sessions,
	}
}

func validateSendTransactionParams(rawParams jsonrpc.Params) (ClientParsedSendTransactionParams, error) {
	if rawParams == nil {
		return ClientParsedSendTransactionParams{}, ErrParamsRequired
	}

	params := ClientSendTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ClientParsedSendTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return ClientParsedSendTransactionParams{}, ErrConnectionTokenIsRequired
	}

	if params.PublicKey == "" {
		return ClientParsedSendTransactionParams{}, ErrPublicKeyIsRequired
	}

	if params.SendingMode == "" {
		return ClientParsedSendTransactionParams{}, ErrSendingModeIsRequired
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
		return ClientParsedSendTransactionParams{}, fmt.Errorf("the sending mode %q is not a valid one", params.SendingMode)
	}

	if sendingMode == apipb.SubmitTransactionRequest_TYPE_UNSPECIFIED {
		return ClientParsedSendTransactionParams{}, ErrSendingModeCannotBeTypeUnspecified
	}

	if params.EncodedTransaction == "" {
		return ClientParsedSendTransactionParams{}, ErrEncodedTransactionIsRequired
	}

	tx, err := base64.StdEncoding.DecodeString(params.EncodedTransaction)
	if err != nil {
		return ClientParsedSendTransactionParams{}, ErrEncodedTransactionIsNotValidBase64String
	}

	return ClientParsedSendTransactionParams{
		Token:          params.Token,
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
		SendingMode:    sendingMode,
	}, nil
}
