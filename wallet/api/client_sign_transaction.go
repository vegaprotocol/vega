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

const TransactionSuccessfullySigned = "The transaction has been successfully signed."

type ClientSignTransactionParams struct {
	PublicKey   string      `json:"publicKey"`
	Transaction interface{} `json:"transaction"`
}

type ClientParsedSignTransactionParams struct {
	PublicKey      string
	RawTransaction string
}

type ClientSignTransactionResult struct {
	Tx *commandspb.Transaction `json:"transaction"`
}

type ClientSignTransaction struct {
	walletStore  WalletStore
	interactor   Interactor
	nodeSelector node.Selector
	spam         SpamHandler
}

func (h *ClientSignTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params, connectedWallet ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := jsonrpc.TraceIDFromContext(ctx)

	params, err := validateSignTransactionParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(strings.NewReader(params.RawTransaction), request); err != nil {
		return nil, invalidParams(fmt.Errorf("the transaction is not a valid Vega command: %w", err))
	}

	if !connectedWallet.CanUseKey(params.PublicKey) {
		return nil, requestNotPermittedError(ErrPublicKeyIsNotAllowedToBeUsed)
	}

	w, err := h.walletStore.GetWallet(ctx, connectedWallet.Name())
	if err != nil {
		if errors.Is(err, ErrWalletIsLocked) {
			h.interactor.NotifyError(ctx, traceID, ApplicationError, err)
		} else {
			h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not retrieve the wallet associated to the connection: %w", err))
		}
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	request.PubKey = params.PublicKey
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, invalidParams(errs)
	}

	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID); err != nil {
		return nil, requestNotPermittedError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	if connectedWallet.RequireInteraction() {
		receivedAt := time.Now()
		approved, err := h.interactor.RequestTransactionReviewForSigning(ctx, traceID, connectedWallet.Hostname(), connectedWallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
		if err != nil {
			if errDetails := handleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
				return nil, errDetails
			}
			h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the transaction review failed: %w", err))
			return nil, internalError(ErrCouldNotSignTransaction)
		}
		if !approved {
			return nil, userRejectionError(ErrUserRejectedSigningOfTransaction)
		}
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Looking for a healthy node...")
	currentNode, err := h.nodeSelector.Node(ctx, func(reportType node.ReportType, msg string) {
		h.interactor.Log(ctx, traceID, LogType(reportType), msg)
	})
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not find a healthy node: %w", err))
		return nil, nodeCommunicationError(ErrNoHealthyNodeAvailable)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Retrieving latest block information...")
	stats, err := currentNode.SpamStatistics(ctx, request.PubKey)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not get the latest block from the node: %w", err))
		return nil, nodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "Latest block information has been retrieved.")

	if stats.LastBlockHeight == 0 {
		h.interactor.NotifyError(ctx, traceID, NetworkError, ErrCouldNotGetLastBlockInformation)
		return nil, nodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}

	if stats.ChainID == "" {
		h.interactor.NotifyError(ctx, traceID, NetworkError, ErrCouldNotGetChainIDFromNode)
		return nil, nodeCommunicationError(ErrCouldNotGetChainIDFromNode)
	}

	err = h.spam.CheckSubmission(request, &stats)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, ApplicationError, fmt.Errorf("could not send transaction: %w", err))
		return nil, applicationCancellationError(err)
	}

	// Sign the payload.
	inputData, err := wcommands.ToMarshaledInputData(request, stats.LastBlockHeight)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not marshal input data: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Signing the transaction...")
	signature, err := w.SignTx(params.PublicKey, commands.BundleInputDataForSigning(inputData, stats.ChainID))
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not sign the transaction: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
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
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not compute the proof-of-work: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "The proof-of-work has been computed.")

	h.interactor.NotifySuccessfulRequest(ctx, traceID, TransactionSuccessfullySigned)

	return ClientSignTransactionResult{
		Tx: tx,
	}, nil
}

func validateSignTransactionParams(rawParams jsonrpc.Params) (ClientParsedSignTransactionParams, error) {
	if rawParams == nil {
		return ClientParsedSignTransactionParams{}, ErrParamsRequired
	}

	params := ClientSignTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ClientParsedSignTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.PublicKey == "" {
		return ClientParsedSignTransactionParams{}, ErrPublicKeyIsRequired
	}

	if params.Transaction == nil {
		return ClientParsedSignTransactionParams{}, ErrTransactionIsRequired
	}

	tx, err := json.Marshal(params.Transaction)
	if err != nil {
		return ClientParsedSignTransactionParams{}, ErrTransactionIsNotValidJSON
	}

	return ClientParsedSignTransactionParams{
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
	}, nil
}

func NewClientSignTransaction(walletStore WalletStore, interactor Interactor, nodeSelector node.Selector, proofOfWork SpamHandler) *ClientSignTransaction {
	return &ClientSignTransaction{
		walletStore:  walletStore,
		interactor:   interactor,
		nodeSelector: nodeSelector,
		spam:         proofOfWork,
	}
}
