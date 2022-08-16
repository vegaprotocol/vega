package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"go.uber.org/zap"
)

// WalletStore is the component used to retrieve and update wallets from the
// computer.
//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_store_mock.go -package mocks code.vegaprotocol.io/vega/wallet/api WalletStore
type WalletStore interface {
	WalletExists(ctx context.Context, name string) (bool, error)
	GetWallet(ctx context.Context, name, passphrase string) (wallet.Wallet, error)
	ListWallets(ctx context.Context) ([]string, error)
	SaveWallet(ctx context.Context, w wallet.Wallet, passphrase string) error
}

// Node is the component used to get network information and send transactions.
//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_mock.go -package mocks code.vegaprotocol.io/vega/wallet/api Node
type Node interface {
	Host() string
	Stop() error
	SendTransaction(context.Context, *commandspb.Transaction, apipb.SubmitTransactionRequest_Type) (string, error)
	CheckTransaction(context.Context, *commandspb.Transaction) (*apipb.CheckTransactionResponse, error)
	HealthCheck(context.Context) error
	LastBlock(context.Context) (*apipb.LastBlockHeightResponse, error)
}

// NodeSelector implementing the strategy for node selection.
//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_selector_mock.go -package mocks code.vegaprotocol.io/vega/wallet/api NodeSelector
type NodeSelector interface {
	Node(ctx context.Context) (Node, error)
	Stop()
}

// Pipeline is the component connecting the client front-end and the JSON-RPC API.
// Convention:
//   - Notify* functions do not expect a response.
//   - Request* functions are expecting a client intervention.
//go:generate go run github.com/golang/mock/mockgen -destination mocks/pipeline_mock.go -package mocks code.vegaprotocol.io/vega/wallet/api Pipeline
type Pipeline interface {
	// NotifyError is used to report errors to the client.
	NotifyError(ctx context.Context, traceID string, t ErrorType, err error)

	// RequestWalletConnectionReview is used to trigger a client review of
	// the wallet connection requested by the specified hostname.
	// It returns true if the client approved the wallet connection, false
	// otherwise.
	RequestWalletConnectionReview(ctx context.Context, traceID, hostname string) (bool, error)

	// NotifySuccessfulRequest is used to notify the client the request is
	// successful.
	NotifySuccessfulRequest(ctx context.Context, traceID string)

	// RequestWalletSelection is used to trigger selection of the wallet the
	// client wants to use for the specified hostname.
	RequestWalletSelection(ctx context.Context, traceID, hostname string, availableWallets []string) (SelectedWallet, error)

	// RequestPassphrase is used to request the client to enter the passphrase of
	// the wallet. It's primarily used for request that requires saving changes
	// on it.
	RequestPassphrase(ctx context.Context, traceID, wallet string) (string, error)

	// RequestPermissionsReview is used to trigger a client review of the permissions
	// requested by the specified hostname.
	// It returns true if the client approved the requested permissions, false
	// otherwise.
	RequestPermissionsReview(ctx context.Context, traceID, hostname, wallet string, perms map[string]string) (bool, error)

	// RequestTransactionReview is used to trigger a client review of the
	// transaction a third-party application wants to send.
	// It returns true if the client approved the sending of the transaction,
	// false otherwise.
	RequestTransactionReview(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error)

	// NotifyTransactionStatus is used to report the transaction status once
	// sent.
	NotifyTransactionStatus(ctx context.Context, traceID, txHash, tx string, err error, sentAt time.Time)
}

// ErrorType defines the type of error that is sent to the client, for fine
// grain error management and reporting.
type ErrorType string

var (
	// InternalError defines an unexpected technical error upon which the client
	// can't act.
	// The client front-end should report it to the client and automatically
	// abort the processing of the ongoing request.
	// It can be raised if a file is not accessible or corrupt, for example.
	InternalError ErrorType = "Internal Error"
	// ServerError defines a programmatic error threw by the server, such as
	// a request cancellation.
	// It's a type of error that should be expected and handled.
	ServerError ErrorType = "Server Error"
	// NetworkError defines an error that comes from the network and its nodes.
	NetworkError ErrorType = "Network Error"
	// ClientError defines an error that originated from the client and that
	// requires its intervention to correct it.
	// It can be raised if a passphrase is invalid, for example.
	ClientError ErrorType = "Client Error"
)

// SelectedWallet holds the result of the wallet selection from the client.
type SelectedWallet struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

// RestrictedAPI builds a JSON-RPC API of the wallet with a subset of the requests
// that are intended to be exposed to external services, such as bots, apps,
// scripts.
// The reason is that we don't want external clients to be able to call
// administration capabilities that should only be exposed to the user.
func RestrictedAPI(log *zap.Logger, walletStore WalletStore, pipeline Pipeline, nodeSelector NodeSelector) (*jsonrpc.API, error) {
	sessions := NewSessions()

	walletAPI := jsonrpc.New(log)
	walletAPI.RegisterMethod("connect_wallet", NewConnectWallet(walletStore, pipeline, sessions))
	walletAPI.RegisterMethod("disconnect_wallet", NewDisconnectWallet(sessions))
	walletAPI.RegisterMethod("get_permissions", NewGetPermissions(sessions))
	walletAPI.RegisterMethod("request_permissions", NewRequestPermissions(walletStore, pipeline, sessions))
	walletAPI.RegisterMethod("list_keys", NewListKeys(sessions))
	walletAPI.RegisterMethod("send_transaction", NewSendTransaction(pipeline, nodeSelector, sessions))

	log.Info("restricted JSON-RPC API initialised")

	return walletAPI, nil
}
