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

type AdminSendRawTransactionParams struct {
	Network                string        `json:"network"`
	NodeAddress            string        `json:"nodeAddress"`
	Retries                uint64        `json:"retries"`
	MaximumRequestDuration time.Duration `json:"maximumRequestDuration"`
	SendingMode            string        `json:"sendingMode"`
	EncodedTransaction     string        `json:"encodedTransaction"`
}

type ParsedAdminSendRawTransactionParams struct {
	Network                string
	NodeAddress            string
	Retries                uint64
	MaximumRequestDuration time.Duration
	SendingMode            apipb.SubmitTransactionRequest_Type
	RawTransaction         string
}

type AdminSendRawTransactionResult struct {
	ReceivedAt time.Time                         `json:"receivedAt"`
	SentAt     time.Time                         `json:"sentAt"`
	TxHash     string                            `json:"transactionHash"`
	Tx         *commandspb.Transaction           `json:"transaction"`
	Node       AdminSendRawTransactionNodeResult `json:"node"`
}

type AdminSendRawTransactionNodeResult struct {
	Host string `json:"host"`
}

type AdminSendRawTransaction struct {
	networkStore        NetworkStore
	nodeSelectorBuilder NodeSelectorBuilder
}

func (h *AdminSendRawTransaction) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminSendRawTransactionParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	receivedAt := time.Now()

	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal([]byte(params.RawTransaction), tx); err != nil {
		return nil, invalidParams(ErrRawTransactionIsNotValidVegaTransaction)
	}

	hosts := []string{params.NodeAddress}
	if len(params.Network) != 0 {
		exists, err := h.networkStore.NetworkExists(params.Network)
		if err != nil {
			return nil, internalError(fmt.Errorf("could not determine if the network exists: %w", err))
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
	}

	nodeSelector, err := h.nodeSelectorBuilder(hosts, params.Retries, params.MaximumRequestDuration)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not initialize the node selector: %w", err))
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

	return AdminSendRawTransactionResult{
		ReceivedAt: receivedAt,
		SentAt:     sentAt,
		TxHash:     txHash,
		Tx:         tx,
		Node: AdminSendRawTransactionNodeResult{
			Host: currentNode.Host(),
		},
	}, nil
}

func NewAdminSendRawTransaction(networkStore NetworkStore, nodeSelectorBuilder NodeSelectorBuilder) *AdminSendRawTransaction {
	return &AdminSendRawTransaction{
		networkStore:        networkStore,
		nodeSelectorBuilder: nodeSelectorBuilder,
	}
}

func validateAdminSendRawTransactionParams(rawParams jsonrpc.Params) (ParsedAdminSendRawTransactionParams, error) {
	if rawParams == nil {
		return ParsedAdminSendRawTransactionParams{}, ErrParamsRequired
	}

	params := AdminSendRawTransactionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ParsedAdminSendRawTransactionParams{}, ErrParamsDoNotMatch
	}

	if params.Network == "" && params.NodeAddress == "" {
		return ParsedAdminSendRawTransactionParams{}, ErrNetworkOrNodeAddressIsRequired
	}

	if params.Network != "" && params.NodeAddress != "" {
		return ParsedAdminSendRawTransactionParams{}, ErrSpecifyingNetworkAndNodeAddressIsNotSupported
	}

	if params.SendingMode == "" {
		return ParsedAdminSendRawTransactionParams{}, ErrSendingModeIsRequired
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
		return ParsedAdminSendRawTransactionParams{}, fmt.Errorf("the sending mode %q is not a valid one", params.SendingMode)
	}

	if sendingMode == apipb.SubmitTransactionRequest_TYPE_UNSPECIFIED {
		return ParsedAdminSendRawTransactionParams{}, ErrSendingModeCannotBeTypeUnspecified
	}

	if params.EncodedTransaction == "" {
		return ParsedAdminSendRawTransactionParams{}, ErrEncodedTransactionIsRequired
	}

	tx, err := base64.StdEncoding.DecodeString(params.EncodedTransaction)
	if err != nil {
		return ParsedAdminSendRawTransactionParams{}, ErrEncodedTransactionIsNotValidBase64String
	}

	return ParsedAdminSendRawTransactionParams{
		Network:                params.Network,
		NodeAddress:            params.NodeAddress,
		RawTransaction:         string(tx),
		SendingMode:            sendingMode,
		Retries:                params.Retries,
		MaximumRequestDuration: params.MaximumRequestDuration,
	}, nil
}
