package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/libs/proto"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/mitchellh/mapstructure"
)

type AdminSendTransactionParams struct {
	Network            string `json:"network"`
	NodeAddress        string `json:"nodeAddress"`
	Retries            uint64 `json:"retries"`
	SendingMode        string `json:"sendingMode"`
	EncodedTransaction string `json:"encodedTransaction"`
}

type ParsedAdminSendTransactionParams struct {
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
	networkStore        NetworkStore
	nodeSelectorBuilder NodeSelectorBuilder
}

func (h *AdminSendTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminSendTransactionParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	receivedAt := time.Now()

	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal([]byte(params.RawTransaction), tx); err != nil {
		return nil, invalidParams(ErrTransactionIsMalformed)
	}

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
			return nil, internalError(ErrNetworkConfigurationDoesNotHaveGRPCNodes)
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

	currentNode, err := nodeSelector.Node(ctx, noNodeSelectionReporting)
	if err != nil {
		return nil, nodeCommunicationError(ErrNoHealthyNodeAvailable)
	}

	sentAt := time.Now()
	txHash, err := currentNode.SendTransaction(ctx, tx, params.SendingMode)
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

func NewAdminSendTransaction(networkStore NetworkStore, nodeSelectorBuilder NodeSelectorBuilder) *AdminSendTransaction {
	return &AdminSendTransaction{
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

	if params.Network == "" && params.NodeAddress == "" {
		return ParsedAdminSendTransactionParams{}, ErrNetworkOrNodeAddressIsRequired
	}

	if params.Network != "" && params.NodeAddress != "" {
		return ParsedAdminSendTransactionParams{}, ErrSpecifyingNetworkAndNodeAddressIsNotSupported
	}

	if params.SendingMode == "" {
		return ParsedAdminSendTransactionParams{}, ErrSendingModeIsRequired
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
		return ParsedAdminSendTransactionParams{}, fmt.Errorf("the sending mode %q is not a valid one", params.SendingMode)
	}

	if sendingMode == apipb.SubmitTransactionRequest_TYPE_UNSPECIFIED {
		return ParsedAdminSendTransactionParams{}, ErrSendingModeCannotBeTypeUnspecified
	}

	if params.EncodedTransaction == "" {
		return ParsedAdminSendTransactionParams{}, ErrEncodedTransactionIsRequired
	}

	tx, err := base64.StdEncoding.DecodeString(params.EncodedTransaction)
	if err != nil {
		return ParsedAdminSendTransactionParams{}, ErrEncodedTransactionIsNotValidBase64String
	}

	return ParsedAdminSendTransactionParams{
		Network:        params.Network,
		NodeAddress:    params.NodeAddress,
		RawTransaction: string(tx),
		SendingMode:    sendingMode,
		Retries:        params.Retries,
	}, nil
}
