package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/commands"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/protobuf/proto"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/mapstructure"
)

type AdminLastBlockData struct {
	ChainID                 string `json:"chainID"`
	BlockHeight             uint64 `json:"blockHeight"`
	BlockHash               string `json:"blockHash"`
	ProofOfWorkHashFunction string `json:"proofOfWorkHashFunction"`
	ProofOfWorkDifficulty   uint32 `json:"proofOfWorkDifficulty"`
}

type AdminSignTransactionParams struct {
	Wallet        string              `json:"wallet"`
	Passphrase    string              `json:"passphrase"`
	PublicKey     string              `json:"publicKey"`
	Transaction   interface{}         `json:"transaction"`
	Network       string              `json:"network"`
	LastBlockData *AdminLastBlockData `json:"lastBlockData"`
}

type ParsedAdminSignTransactionParams struct {
	Wallet         string
	Passphrase     string
	PublicKey      string
	RawTransaction string
	Network        string
	LastBlockData  *AdminLastBlockData
}

type AdminSignTransactionResult struct {
	Tx                 *commandspb.Transaction `json:"transaction"`
	EncodedTransaction string
}

type AdminSignTransaction struct {
	walletStore         WalletStore
	networkStore        NetworkStore
	nodeSelectorBuilder NodeSelectorBuilder
}

func (h *AdminSignTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminSignTransactionParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	if err := h.walletStore.UnlockWallet(ctx, params.Wallet, params.Passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return nil, invalidParams(err)
		}
		return nil, internalError(fmt.Errorf("could not unlock the wallet: %w", err))
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.Unmarshal(strings.NewReader(params.RawTransaction), request); err != nil {
		return nil, invalidParams(ErrTransactionIsNotValidVegaCommand)
	}

	request.PubKey = params.PublicKey
	request.Propagate = true
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, invalidParams(errs)
	}

	if params.Network != "" {
		lastBlockData, errDetails := h.getLastBlockDataFromNetwork(ctx, params)
		if errDetails != nil {
			return nil, errDetails
		}
		params.LastBlockData = lastBlockData
	}

	marshaledInputData, err := wcommands.ToMarshaledInputData(request, params.LastBlockData.BlockHeight)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not marshal the input data: %w", err))
	}

	signature, err := w.SignTx(params.PublicKey, commands.BundleInputDataForSigning(marshaledInputData, params.LastBlockData.ChainID))
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
	powNonce, _, err := vgcrypto.PoW(params.LastBlockData.BlockHash, txID, uint(params.LastBlockData.ProofOfWorkDifficulty), params.LastBlockData.ProofOfWorkHashFunction)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not compute the proof-of-work: %w", err))
	}
	tx.Pow = &commandspb.ProofOfWork{
		Nonce: powNonce,
		Tid:   txID,
	}

	rawTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not marshal the transaction: %w", err))
	}

	return AdminSignTransactionResult{
		Tx:                 tx,
		EncodedTransaction: base64.StdEncoding.EncodeToString(rawTx),
	}, nil
}

func (h *AdminSignTransaction) getLastBlockDataFromNetwork(ctx context.Context, params ParsedAdminSignTransactionParams) (*AdminLastBlockData, *jsonrpc.ErrorDetails) {
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

	nodeSelector, err := h.nodeSelectorBuilder(n.API.GRPC.Hosts, n.API.GRPC.Retries)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not initializing the node selector: %w", err))
	}

	node, err := nodeSelector.Node(ctx, noNodeSelectionReporting)
	if err != nil {
		return nil, nodeCommunicationError(ErrNoHealthyNodeAvailable)
	}

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

func NewAdminSignTransaction(walletStore WalletStore, networkStore NetworkStore, nodeSelectorBuilder NodeSelectorBuilder) *AdminSignTransaction {
	return &AdminSignTransaction{
		walletStore:         walletStore,
		networkStore:        networkStore,
		nodeSelectorBuilder: nodeSelectorBuilder,
	}
}

func validateAdminSignTransactionParams(rawParams jsonrpc.Params) (ParsedAdminSignTransactionParams, error) {
	if rawParams == nil {
		return ParsedAdminSignTransactionParams{}, ErrParamsRequired
	}

	params := AdminSignTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ParsedAdminSignTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return ParsedAdminSignTransactionParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return ParsedAdminSignTransactionParams{}, ErrPassphraseIsRequired
	}

	if params.PublicKey == "" {
		return ParsedAdminSignTransactionParams{}, ErrPublicKeyIsRequired
	}

	if params.Transaction == nil || params.Transaction == "" {
		return ParsedAdminSignTransactionParams{}, ErrTransactionIsRequired
	}

	tx, err := json.Marshal(params.Transaction)
	if err != nil {
		return ParsedAdminSignTransactionParams{}, ErrTransactionIsNotValidJSON
	}

	if params.Network != "" && params.LastBlockData != nil {
		return ParsedAdminSignTransactionParams{}, ErrSpecifyingNetworkAndLastBlockDataIsNotSupported
	}

	if params.Network == "" && params.LastBlockData == nil {
		return ParsedAdminSignTransactionParams{}, ErrLastBlockDataOrNetworkIsRequired
	}

	if params.LastBlockData != nil {
		if params.LastBlockData.BlockHeight == 0 {
			return ParsedAdminSignTransactionParams{}, ErrBlockHeightIsRequired
		}
		if params.LastBlockData.ChainID == "" {
			return ParsedAdminSignTransactionParams{}, ErrChainIDIsRequired
		}
		if params.LastBlockData.BlockHash == "" {
			return ParsedAdminSignTransactionParams{}, ErrBlockHashIsRequired
		}
		if params.LastBlockData.ProofOfWorkDifficulty == 0 {
			return ParsedAdminSignTransactionParams{}, ErrProofOfWorkDifficultyRequired
		}
		if params.LastBlockData.ProofOfWorkHashFunction == "" {
			return ParsedAdminSignTransactionParams{}, ErrProofOfWorkHashFunctionRequired
		}
	}

	return ParsedAdminSignTransactionParams{
		Wallet:         params.Wallet,
		Passphrase:     params.Passphrase,
		PublicKey:      params.PublicKey,
		RawTransaction: string(tx),
		Network:        params.Network,
		LastBlockData:  params.LastBlockData,
	}, nil
}
