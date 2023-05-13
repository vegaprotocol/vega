package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/commands"
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
	PublicKey   string      `json:"publicKey"`
	SendingMode string      `json:"sendingMode"`
	Transaction interface{} `json:"transaction"`
}

type ClientParsedSendTransactionParams struct {
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
	walletStore  WalletStore
	interactor   Interactor
	nodeSelector node.Selector
	spam         SpamHandler
}

func (h *ClientSendTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params, connectedWallet ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := jsonrpc.TraceIDFromContext(ctx)

	params, err := validateSendTransactionParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	txReader := strings.NewReader(params.RawTransaction)
	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(txReader, request); err != nil {
		return nil, InvalidParams(fmt.Errorf("the transaction does not use a valid Vega command: %w", err))
	}

	if !connectedWallet.CanUseKey(params.PublicKey) {
		return nil, RequestNotPermittedError(ErrPublicKeyIsNotAllowedToBeUsed)
	}

	w, err := h.walletStore.GetWallet(ctx, connectedWallet.Name())
	if err != nil {
		if errors.Is(err, ErrWalletIsLocked) {
			h.interactor.NotifyError(ctx, traceID, ApplicationErrorType, err)
		} else {
			h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not retrieve the wallet associated to the connection: %w", err))
		}
		return nil, InternalError(ErrCouldNotSignTransaction)
	}

	request.PubKey = params.PublicKey
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, InvalidParams(errs)
	}

	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID, TransactionReviewWorkflow, 2); err != nil {
		return nil, RequestNotPermittedError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	receivedAt := time.Now()
	if connectedWallet.RequireInteraction() {
		approved, err := h.interactor.RequestTransactionReviewForSending(ctx, traceID, 1, connectedWallet.Hostname(), connectedWallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
		if err != nil {
			if errDetails := HandleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
				return nil, errDetails
			}
			h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("requesting the transaction review failed: %w", err))
			return nil, InternalError(ErrCouldNotSendTransaction)
		}
		if !approved {
			return nil, UserRejectionError(ErrUserRejectedSendingOfTransaction)
		}
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Looking for a healthy node...")
	currentNode, err := h.nodeSelector.Node(ctx, func(reportType node.ReportType, msg string) {
		h.interactor.Log(ctx, traceID, LogType(reportType), msg)
	})
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, NetworkErrorType, fmt.Errorf("could not find a healthy node: %w", err))
		return nil, NodeCommunicationError(ErrNoHealthyNodeAvailable)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Retrieving latest block information...")
	stats, err := currentNode.SpamStatistics(ctx, request.PubKey)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, NetworkErrorType, fmt.Errorf("could not get the latest block from node: %w", err))
		return nil, NodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "Latest block information has been retrieved.")

	if stats.LastBlockHeight == 0 {
		h.interactor.NotifyError(ctx, traceID, NetworkErrorType, ErrCouldNotGetLastBlockInformation)
		return nil, NodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}

	if stats.ChainID == "" {
		h.interactor.NotifyError(ctx, traceID, NetworkErrorType, ErrCouldNotGetChainIDFromNode)
		return nil, NodeCommunicationError(ErrCouldNotGetChainIDFromNode)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Verifying if the transaction passes the anti-spam rules...")
	err = h.spam.CheckSubmission(request, &stats)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, ApplicationErrorType, fmt.Errorf("could not send transaction: %w", err))
		return nil, ApplicationCancellationError(err)
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "The transaction passes the anti-spam rules.")

	// Sign the payload.
	rawInputData := wcommands.ToInputData(request, stats.LastBlockHeight)
	inputData, err := commands.MarshalInputData(rawInputData)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not marshal input data: %w", err))
		return nil, InternalError(ErrCouldNotSendTransaction)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Signing the transaction...")
	signature, err := w.SignTx(params.PublicKey, commands.BundleInputDataForSigning(inputData, stats.ChainID))
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not sign the transaction: %w", err))
		return nil, InternalError(ErrCouldNotSendTransaction)
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
	tx.Pow, err = h.spam.GenerateProofOfWork(params.PublicKey, &stats)
	if err != nil {
		if errors.Is(err, ErrTransactionsPerBlockLimitReached) || errors.Is(err, ErrBlockHeightTooHistoric) {
			h.interactor.NotifyError(ctx, traceID, ApplicationErrorType, fmt.Errorf("could not compute the proof-of-work: %w", err))
			return nil, ApplicationCancellationError(err)
		}
		h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not compute the proof-of-work: %w", err))
		return nil, InternalError(ErrCouldNotSendTransaction)
	}

	h.interactor.Log(ctx, traceID, SuccessLog, "The proof-of-work has been computed.")
	sentAt := time.Now()

	h.interactor.Log(ctx, traceID, InfoLog, "Sending the transaction to the network...")
	txHash, err := currentNode.SendTransaction(ctx, tx, params.SendingMode)
	if err != nil {
		h.interactor.NotifyFailedTransaction(ctx, traceID, 2, protoToJSON(rawInputData), protoToJSON(tx), err, sentAt, currentNode.Host())
		return nil, NetworkErrorFromTransactionError(err)
	}

	h.interactor.NotifySuccessfulTransaction(ctx, traceID, 2, txHash, protoToJSON(rawInputData), protoToJSON(tx), sentAt, currentNode.Host())

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

func validateSendTransactionParams(rawParams jsonrpc.Params) (ClientParsedSendTransactionParams, error) {
	if rawParams == nil {
		return ClientParsedSendTransactionParams{}, ErrParamsRequired
	}

	params := ClientSendTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ClientParsedSendTransactionParams{}, ErrParamsDoNotMatch
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

	if params.Transaction == nil {
		return ClientParsedSendTransactionParams{}, ErrTransactionIsRequired
	}

	tx, err := json.Marshal(params.Transaction)
	if err != nil {
		return ClientParsedSendTransactionParams{}, ErrTransactionIsNotValidJSON
	}

	return ClientParsedSendTransactionParams{
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
		SendingMode:    sendingMode,
	}, nil
}

func NewClientSendTransaction(walletStore WalletStore, interactor Interactor, nodeSelector node.Selector, pow SpamHandler) *ClientSendTransaction {
	return &ClientSendTransaction{
		walletStore:  walletStore,
		interactor:   interactor,
		nodeSelector: nodeSelector,
		spam:         pow,
	}
}
