package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/session"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/mapstructure"
)

const TransactionSuccessfullySigned = "The transaction has been successfully signed."

type ClientSignTransactionParams struct {
	Token       string      `json:"token"`
	PublicKey   string      `json:"publicKey"`
	Transaction interface{} `json:"transaction"`
}

type ClientParsedSignTransactionParams struct {
	Token          string
	PublicKey      string
	RawTransaction string
}

type ClientSignTransactionResult struct {
	Tx *commandspb.Transaction `json:"transaction"`
}

type ClientSignTransaction struct {
	interactor   Interactor
	nodeSelector node.Selector
	pow          ProofOfWork
	sessions     *session.Sessions
	time         TimeProvider
}

func (h *ClientSignTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params, metadata jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateSignTransactionParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(strings.NewReader(params.RawTransaction), request); err != nil {
		return nil, invalidParams(ErrTransactionIsNotValidVegaCommand)
	}

	connectedWallet, err := h.sessions.GetConnectedWallet(params.Token, h.time.Now())
	if err != nil {
		return nil, invalidParams(err)
	}

	if !connectedWallet.CanUseKey(params.PublicKey) {
		return nil, requestNotPermittedError(ErrPublicKeyIsNotAllowedToBeUsed)
	}

	request.PubKey = params.PublicKey
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, invalidParams(errs)
	}

	if err := h.interactor.NotifyInteractionSessionBegan(ctx, metadata.TraceID); err != nil {
		return nil, internalError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, metadata.TraceID)

	if connectedWallet.RequireInteraction() {
		receivedAt := time.Now()
		approved, err := h.interactor.RequestTransactionReviewForSigning(ctx, metadata.TraceID, connectedWallet.Hostname, connectedWallet.Wallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
		if err != nil {
			if errDetails := handleRequestFlowError(ctx, metadata.TraceID, h.interactor, err); errDetails != nil {
				return nil, errDetails
			}
			h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("requesting the transaction review failed: %w", err))
			return nil, internalError(ErrCouldNotSignTransaction)
		}
		if !approved {
			return nil, userRejectionError()
		}
	}

	h.interactor.Log(ctx, metadata.TraceID, InfoLog, "Looking for a healthy node...")
	currentNode, err := h.nodeSelector.Node(ctx, func(reportType node.ReportType, msg string) {
		h.interactor.Log(ctx, metadata.TraceID, LogType(reportType), msg)
	})
	if err != nil {
		h.interactor.NotifyError(ctx, metadata.TraceID, NetworkError, fmt.Errorf("could not find a healthy node: %w", err))
		return nil, nodeCommunicationError(ErrNoHealthyNodeAvailable)
	}

	h.interactor.Log(ctx, metadata.TraceID, InfoLog, "Retrieving latest block information...")
	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		h.interactor.NotifyError(ctx, metadata.TraceID, NetworkError, fmt.Errorf("could not get the latest block from the node: %w", err))
		return nil, nodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}
	h.interactor.Log(ctx, metadata.TraceID, SuccessLog, "Latest block information has been retrieved.")

	if lastBlockData.ChainID == "" {
		h.interactor.NotifyError(ctx, metadata.TraceID, NetworkError, ErrCouldNotGetChainIDFromNode)
		return nil, nodeCommunicationError(ErrCouldNotGetChainIDFromNode)
	}

	// Sign the payload.
	inputData, err := wcommands.ToMarshaledInputData(request, lastBlockData.BlockHeight)
	if err != nil {
		h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("could not marshal input data: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	h.interactor.Log(ctx, metadata.TraceID, InfoLog, "Signing the transaction...")
	signature, err := connectedWallet.Wallet.SignTx(params.PublicKey, commands.BundleInputDataForSigning(inputData, lastBlockData.ChainID))
	if err != nil {
		h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("could not sign command: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}
	h.interactor.Log(ctx, metadata.TraceID, SuccessLog, "The transaction has been signed.")

	// Build the transaction.
	tx := commands.NewTransaction(params.PublicKey, inputData, &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	})

	// Generate the proof of work for the transaction.
	h.interactor.Log(ctx, metadata.TraceID, InfoLog, "Computing proof-of-work...")
	tx.Pow, err = h.pow.Generate(params.PublicKey, &lastBlockData)
	if err != nil {
		h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("could not compute the proof-of-work: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	h.interactor.Log(ctx, metadata.TraceID, SuccessLog, "The proof-of-work has been computed.")

	h.interactor.NotifySuccessfulRequest(ctx, metadata.TraceID, TransactionSuccessfullySigned)

	return ClientSignTransactionResult{
		Tx: tx,
	}, nil
}

func NewSignTransaction(interactor Interactor, nodeSelector node.Selector, pow ProofOfWork, sessions *session.Sessions, tp ...TimeProvider) *ClientSignTransaction {
	return &ClientSignTransaction{
		interactor:   interactor,
		nodeSelector: nodeSelector,
		pow:          pow,
		sessions:     sessions,
		time:         extractTimeProvider(tp...),
	}
}

func validateSignTransactionParams(rawParams jsonrpc.Params) (ClientParsedSignTransactionParams, error) {
	if rawParams == nil {
		return ClientParsedSignTransactionParams{}, ErrParamsRequired
	}

	params := ClientSignTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ClientParsedSignTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return ClientParsedSignTransactionParams{}, ErrConnectionTokenIsRequired
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
		Token:          params.Token,
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
	}, nil
}
