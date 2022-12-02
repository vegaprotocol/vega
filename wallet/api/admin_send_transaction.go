package api

import (
	"context"
	"encoding/json"
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
	"github.com/mitchellh/mapstructure"
)

type AdminSendTransactionParams struct {
	Wallet      string      `json:"wallet"`
	Passphrase  string      `json:"passphrase"`
	PublicKey   string      `json:"publicKey"`
	Network     string      `json:"network"`
	NodeAddress string      `json:"nodeAddress"`
	Retries     uint64      `json:"retries"`
	SendingMode string      `json:"sendingMode"`
	Transaction interface{} `json:"transaction"`
}

type ParsedAdminSendTransactionParams struct {
	Wallet         string
	Passphrase     string
	PublicKey      string
	Network        string
	NodeAddress    string
	Retries        uint64
	SendingMode    apipb.SubmitTransactionRequest_Type
	RawTransaction string
}

type AdminSendTransactionResult struct {
	ReceivedAt time.Time               `json:"receivedAt"`
	SentAt     time.Time               `json:"sentAt"`
	TxHash     string                  `json:"transactionHash"`
	Tx         *commandspb.Transaction `json:"transaction"`
}

type AdminSendTransaction struct {
	walletStore         WalletStore
	networkStore        NetworkStore
	nodeSelectorBuilder NodeSelectorBuilder
}

func (h *AdminSendTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminSendTransactionParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	receivedAt := time.Now()

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet, params.Passphrase)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(strings.NewReader(params.RawTransaction), request); err != nil {
		return nil, invalidParams(ErrTransactionIsMalformed)
	}

	request.PubKey = params.PublicKey
	request.Propagate = true
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, invalidParams(errs)
	}

	node, errDetails := h.getNode(ctx, params)
	if errDetails != nil {
		return nil, errDetails
	}

	lastBlockData, errDetails := h.getLastBlockDataFromNetwork(ctx, node)
	if err != nil {
		return nil, errDetails
	}

	marshaledInputData, err := wcommands.ToMarshaledInputData(request, lastBlockData.BlockHeight)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not marshal the input data: %w", err))
	}

	signature, err := w.SignTx(params.PublicKey, commands.BundleInputDataForSigning(marshaledInputData, lastBlockData.ChainID))
	if err != nil {
		return nil, internalError(fmt.Errorf("could not sign the transaction: %w", err))
	}

	// Build the transaction.
	tx := commands.NewTransaction(params.PublicKey, marshaledInputData, &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	})

	// Generate the proof of work for the transaction.
	txID := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(lastBlockData.BlockHash, txID, uint(lastBlockData.ProofOfWorkDifficulty), lastBlockData.ProofOfWorkHashFunction)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not compute the proof-of-work: %w", err))
	}

	tx.Pow = &commandspb.ProofOfWork{
		Nonce: powNonce,
		Tid:   txID,
	}

	sentAt := time.Now()
	txHash, err := node.SendTransaction(ctx, tx, params.SendingMode)
	if err != nil {
		return nil, networkErrorFromTransactionError(err)
	}

	return AdminSendTransactionResult{
		ReceivedAt: receivedAt,
		SentAt:     sentAt,
		TxHash:     txHash,
		Tx:         tx,
	}, nil
}

func (h *AdminSendTransaction) getNode(ctx context.Context, params ParsedAdminSendTransactionParams) (node.Node, *jsonrpc.ErrorDetails) {
	var hosts []string
	var retries uint64
	if len(params.Network) != 0 {
		exists, err := h.networkStore.NetworkExists(params.Network)
		if err != nil {
			return nil, internalError(fmt.Errorf("could not check the network existence: %w", err))
		} else if !exists {
			return nil, invalidParams(ErrNetworkDoesNotExist)
		}

		n, err := h.networkStore.GetNetwork(params.Network)
		if err != nil {
			return nil, internalError(fmt.Errorf("could not retrieve the network configuration: %w", err))
		}

		if err := n.EnsureCanConnectGRPCNode(); err != nil {
			return nil, invalidParams(ErrNetworkConfigurationDoesNotHaveGRPCNodes)
		}
		hosts = n.API.GRPC.Hosts
		retries = n.API.GRPC.Retries
	} else {
		hosts = []string{params.NodeAddress}
		retries = params.Retries
	}

	nodeSelector, err := h.nodeSelectorBuilder(hosts, retries)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not initializing the node selector: %w", err))
	}

	node, err := nodeSelector.Node(ctx, noNodeSelectionReporting)
	if err != nil {
		return nil, nodeCommunicationError(ErrNoHealthyNodeAvailable)
	}
	return node, nil
}

func (h *AdminSendTransaction) getLastBlockDataFromNetwork(ctx context.Context, node node.Node) (*AdminLastBlockData, *jsonrpc.ErrorDetails) {
	lastBlock, err := node.LastBlock(ctx)
	if err != nil {
		return nil, nodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}

	if lastBlock.ChainID == "" {
		return nil, nodeCommunicationError(ErrCouldNotGetChainIDFromNode)
	}

	return &AdminLastBlockData{
		BlockHash:               lastBlock.BlockHash,
		ChainID:                 lastBlock.ChainID,
		BlockHeight:             lastBlock.BlockHeight,
		ProofOfWorkHashFunction: lastBlock.ProofOfWorkHashFunction,
		ProofOfWorkDifficulty:   lastBlock.ProofOfWorkDifficulty,
	}, nil
}

func NewAdminSendTransaction(walletStore WalletStore, networkStore NetworkStore, nodeSelectorBuilder NodeSelectorBuilder) *AdminSendTransaction {
	return &AdminSendTransaction{
		walletStore:         walletStore,
		networkStore:        networkStore,
		nodeSelectorBuilder: nodeSelectorBuilder,
	}
}

func validateAdminSendTransactionParams(rawParams jsonrpc.Params) (ParsedAdminSendTransactionParams, error) {
	if rawParams == nil {
		return ParsedAdminSendTransactionParams{}, ErrParamsRequired
	}

	params := AdminSendTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ParsedAdminSendTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return ParsedAdminSendTransactionParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return ParsedAdminSendTransactionParams{}, ErrPassphraseIsRequired
	}

	if params.PublicKey == "" {
		return ParsedAdminSendTransactionParams{}, ErrPublicKeyIsRequired
	}

	if params.Network == "" && params.NodeAddress == "" {
		return ParsedAdminSendTransactionParams{}, ErrNetworkOrNodeAddressIsRequired
	}

	if params.Network != "" && params.NodeAddress != "" {
		return ParsedAdminSendTransactionParams{}, ErrSpecifyingNetworkAndNodeAddressIsNotSupported
	}

	if params.Transaction == nil || params.Transaction == "" {
		return ParsedAdminSendTransactionParams{}, ErrTransactionIsRequired
	}

	tx, err := json.Marshal(params.Transaction)
	if err != nil {
		return ParsedAdminSendTransactionParams{}, ErrEncodedTransactionIsNotValid
	}

	return ParsedAdminSendTransactionParams{
		Wallet:         params.Wallet,
		Passphrase:     params.Passphrase,
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
		Network:        params.Network,
		NodeAddress:    params.NodeAddress,
		Retries:        params.Retries,
	}, nil
}
