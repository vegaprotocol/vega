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
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/api/node"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/mapstructure"
)

const TransactionSuccessfullyChecked = "The transaction has been successfully checked."

type ClientCheckTransactionParams struct {
	PublicKey   string      `json:"publicKey"`
	Transaction interface{} `json:"transaction"`
}

type ClientParsedCheckTransactionParams struct {
	PublicKey      string
	RawTransaction string
}

type ClientCheckTransactionResult struct {
	ReceivedAt  time.Time               `json:"receivedAt"`
	SentAt      time.Time               `json:"sentAt"`
	Transaction *commandspb.Transaction `json:"transaction"`
}

type ClientCheckTransaction struct {
	walletStore       WalletStore
	interactor        Interactor
	nodeSelector      node.Selector
	spam              SpamHandler
	requestController *RequestController
}

func (h *ClientCheckTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params, connectedWallet ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := jsonrpc.TraceIDFromContext(ctx)

	receivedAt := time.Now()

	params, err := validateCheckTransactionParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(strings.NewReader(params.RawTransaction), request); err != nil {
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
		return nil, InternalError(ErrCouldNotCheckTransaction)
	}

	request.PubKey = params.PublicKey
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, InvalidParams(errs)
	}

	iAmDone, err := h.requestController.IsPublicKeyAlreadyInUse(params.PublicKey)
	if err != nil {
		return nil, RequestNotPermittedError(err)
	}
	defer iAmDone()

	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID, TransactionReviewWorkflow, 2); err != nil {
		return nil, RequestNotPermittedError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	if connectedWallet.RequireInteraction() {
		approved, err := h.interactor.RequestTransactionReviewForChecking(ctx, traceID, 1, connectedWallet.Hostname(), connectedWallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
		if err != nil {
			if errDetails := HandleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
				return nil, errDetails
			}
			h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("requesting the transaction review failed: %w", err))
			return nil, InternalError(ErrCouldNotCheckTransaction)
		}
		if !approved {
			return nil, UserRejectionError(ErrUserRejectedCheckingOfTransaction)
		}
	} else {
		h.interactor.Log(ctx, traceID, InfoLog, fmt.Sprintf("Trying to check the transaction: %v", request.String()))
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
		h.interactor.NotifyError(ctx, traceID, NetworkErrorType, fmt.Errorf("could not get the latest block information from the node: %w", err))
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
		return nil, InternalError(ErrCouldNotCheckTransaction)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Signing the transaction...")
	signature, err := w.SignTx(params.PublicKey, commands.BundleInputDataForSigning(inputData, stats.ChainID))
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not sign the transaction: %w", err))
		return nil, InternalError(ErrCouldNotCheckTransaction)
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
		return nil, InternalError(ErrCouldNotCheckTransaction)
	}

	h.interactor.Log(ctx, traceID, SuccessLog, "The proof-of-work has been computed.")
	sentAt := time.Now()

	h.interactor.Log(ctx, traceID, InfoLog, "Checking the transaction on the network...")
	if err := currentNode.CheckTransaction(ctx, tx); err != nil {
		h.interactor.NotifyFailedTransaction(ctx, traceID, 2, protoToJSON(rawInputData), protoToJSON(tx), err, sentAt, currentNode.Host())
		return nil, NetworkErrorFromTransactionError(err)
	}

	h.interactor.NotifySuccessfulRequest(ctx, traceID, 2, TransactionSuccessfullyChecked)

	return ClientCheckTransactionResult{
		ReceivedAt:  receivedAt,
		SentAt:      sentAt,
		Transaction: tx,
	}, nil
}

func validateCheckTransactionParams(rawParams jsonrpc.Params) (ClientParsedCheckTransactionParams, error) {
	if rawParams == nil {
		return ClientParsedCheckTransactionParams{}, ErrParamsRequired
	}

	params := ClientCheckTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ClientParsedCheckTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.PublicKey == "" {
		return ClientParsedCheckTransactionParams{}, ErrPublicKeyIsRequired
	}

	if params.Transaction == nil {
		return ClientParsedCheckTransactionParams{}, ErrTransactionIsRequired
	}

	if params.Transaction == nil {
		return ClientParsedCheckTransactionParams{}, ErrTransactionIsRequired
	}

	tx, err := json.Marshal(params.Transaction)
	if err != nil {
		return ClientParsedCheckTransactionParams{}, ErrTransactionIsNotValidJSON
	}

	return ClientParsedCheckTransactionParams{
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
	}, nil
}

func NewClientCheckTransaction(walletStore WalletStore, interactor Interactor, nodeSelector node.Selector, pow SpamHandler, requestController *RequestController) *ClientCheckTransaction {
	return &ClientCheckTransaction{
		walletStore:       walletStore,
		interactor:        interactor,
		nodeSelector:      nodeSelector,
		spam:              pow,
		requestController: requestController,
	}
}
