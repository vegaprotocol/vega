package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

const TransactionSuccessfullySigned = "The transaction has been successfully signed."

type ClientSignTransactionParams struct {
	Token              string      `json:"token"`
	PublicKey          string      `json:"publicKey"`
	EncodedTransaction string      `json:"encodedTransaction"`
	Transaction        interface{} `json:"transaction"`
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
	sessions     *Sessions
}

func (h *ClientSignTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
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

	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID); err != nil {
		return nil, internalError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	receivedAt := time.Now()
	approved, err := h.interactor.RequestTransactionReviewForSigning(ctx, traceID, connectedWallet.Hostname, connectedWallet.Wallet.Name(), params.PublicKey, params.RawTransaction, receivedAt)
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
			return nil, errDetails
		}
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the transaction review failed: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
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
		return nil, nodeCommunicationError(ErrNoHealthyNodeAvailable)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Retrieving latest block information...")
	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, NetworkError, fmt.Errorf("could not get the latest block from the node: %w", err))
		return nil, nodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "Latest block information has been retrieved.")

	if lastBlockData.ChainID == "" {
		h.interactor.NotifyError(ctx, traceID, NetworkError, ErrCouldNotGetChainIDFromNode)
		return nil, nodeCommunicationError(ErrCouldNotGetChainIDFromNode)
	}

	// Sign the payload.
	inputData, err := wcommands.ToMarshaledInputData(request, lastBlockData.BlockHeight)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not marshal input data: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}

	h.interactor.Log(ctx, traceID, InfoLog, "Signing the transaction...")
	signature, err := connectedWallet.Wallet.SignTx(params.PublicKey, commands.BundleInputDataForSigning(inputData, lastBlockData.ChainID))
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not sign command: %w", err))
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
	txID := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(lastBlockData.BlockHash, txID, uint(lastBlockData.ProofOfWorkDifficulty), lastBlockData.ProofOfWorkHashFunction)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not compute the proof-of-work: %w", err))
		return nil, internalError(ErrCouldNotSignTransaction)
	}
	tx.Pow = &commandspb.ProofOfWork{
		Tid:   txID,
		Nonce: powNonce,
	}
	h.interactor.Log(ctx, traceID, SuccessLog, "The proof-of-work has been computed.")

	h.interactor.NotifySuccessfulRequest(ctx, traceID, TransactionSuccessfullySigned)

	return ClientSignTransactionResult{
		Tx: tx,
	}, nil
}

func NewSignTransaction(interactor Interactor, nodeSelector node.Selector, sessions *Sessions) *ClientSignTransaction {
	return &ClientSignTransaction{
		interactor:   interactor,
		nodeSelector: nodeSelector,
		sessions:     sessions,
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

	if params.EncodedTransaction == "" && params.Transaction == nil {
		return ClientParsedSignTransactionParams{}, ErrTransactionIsRequired
	}

	if params.EncodedTransaction != "" && params.Transaction != nil {
		return ClientParsedSignTransactionParams{}, ErrEncodedTransactionAndTransactionSupplied
	}

	var tx []byte
	var err error

	if params.EncodedTransaction != "" {
		tx, err = base64.StdEncoding.DecodeString(params.EncodedTransaction)
		if err != nil {
			return ClientParsedSignTransactionParams{}, ErrEncodedTransactionIsNotValidBase64String
		}
	}

	if params.Transaction != nil {
		tx, err = json.Marshal(params.Transaction)
		if err != nil {
			return ClientParsedSignTransactionParams{}, ErrEncodedTransactionIsNotValid
		}
	}

	return ClientParsedSignTransactionParams{
		Token:          params.Token,
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
	}, nil
}
