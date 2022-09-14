package api

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"go.uber.org/zap"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/wallet/api WalletStore,NetworkStore,Node,NodeSelector,Pipeline

// WalletStore is the component used to retrieve and update wallets from the
// computer.
type WalletStore interface {
	WalletExists(ctx context.Context, name string) (bool, error)
	GetWallet(ctx context.Context, name, passphrase string) (wallet.Wallet, error)
	ListWallets(ctx context.Context) ([]string, error)
	SaveWallet(ctx context.Context, w wallet.Wallet, passphrase string) error
	DeleteWallet(ctx context.Context, name string) error
	GetWalletPath(name string) string
}

// NetworkStore is the component used to retrieve and update the networks from the
// computer.
type NetworkStore interface {
	NetworkExists(string) (bool, error)
	GetNetwork(string) (*network.Network, error)
	SaveNetwork(*network.Network) error
	ListNetworks() ([]string, error)
	GetNetworkPath(string) string
	DeleteNetwork(string) error
}

// Node is the component used to get network information and send transactions.
type Node interface {
	Host() string
	Stop() error
	SendTransaction(context.Context, *commandspb.Transaction, apipb.SubmitTransactionRequest_Type) (string, error)
	CheckTransaction(context.Context, *commandspb.Transaction) (*apipb.CheckTransactionResponse, error)
	HealthCheck(context.Context) error
	LastBlock(context.Context) (*apipb.LastBlockHeightResponse, error)
}

// NodeSelector implementing the strategy for node selection.
type NodeSelector interface {
	Node(ctx context.Context) (Node, error)
	Stop()
}

// Pipeline is the component connecting the wallet front-end and the JSON-RPC API.
// Convention:
//   - Notify* functions do not expect a response.
//   - Request* functions are expecting a user intervention.
type Pipeline interface {
	// NotifyError is used to report errors to the user.
	// This is a terminal message. Nothing should be expected on the request
	// after receiving this.
	NotifyError(ctx context.Context, traceID string, t ErrorType, err error)

	// Log is used to report information of any kind to the user. This is just
	// to log internal activity to provide feedback to the wallet front-end.
	// The log should be displayed without triggering any actions.
	// Receiving a success or an error log shouldn't be confused with the
	// NotifyError and NotifySuccessfulRequest that send terminal messages.
	Log(ctx context.Context, traceID string, t LogType, msg string)

	// RequestWalletConnectionReview is used to trigger a user review of
	// the wallet connection requested by the specified hostname.
	// It returns true if the user approved the wallet connection, false
	// otherwise.
	RequestWalletConnectionReview(ctx context.Context, traceID, hostname string) (bool, error)

	// NotifySuccessfulRequest is used to notify the user the request is
	// successful.
	// This is a terminal message. Nothing should be expected on the request
	// after receiving this.
	NotifySuccessfulRequest(ctx context.Context, traceID string)

	// RequestWalletSelection is used to trigger selection of the wallet the
	// user wants to use for the specified hostname.
	RequestWalletSelection(ctx context.Context, traceID, hostname string, availableWallets []string) (SelectedWallet, error)

	// RequestPassphrase is used to request the user to enter the passphrase of
	// the wallet. It's primarily used for request that requires saving changes
	// on it.
	RequestPassphrase(ctx context.Context, traceID, wallet string) (string, error)

	// RequestPermissionsReview is used to trigger a user review of the permissions
	// requested by the specified hostname.
	// It returns true if the user approved the requested permissions, false
	// otherwise.
	RequestPermissionsReview(ctx context.Context, traceID, hostname, wallet string, perms map[string]string) (bool, error)

	// RequestTransactionSendingReview is used to trigger a user review of the
	// transaction a third-party application wants to send.
	// It returns true if the user approved the sending of the transaction,
	// false otherwise.
	RequestTransactionSendingReview(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error)

	// RequestTransactionSigningReview is used to trigger a user review of the
	// transaction a third-party application wants to sign. The wallet doesn't
	// send the transaction.
	// It returns true if the user approved the signing of the transaction,
	// false otherwise.
	RequestTransactionSigningReview(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error)

	// NotifyTransactionStatus is used to report the transaction status once
	// sent.
	// This is a terminal message. Nothing should be expected on the request
	// after receiving this.
	NotifyTransactionStatus(ctx context.Context, traceID, txHash, tx string, err error, sentAt time.Time)
}

// ErrorType defines the type of error that is sent to the user, for fine
// grain error management and reporting.
type ErrorType string

var (
	// InternalError defines an unexpected technical error upon which the user
	// can't act.
	// The wallet front-end should report it to the user and automatically
	// abort the processing of the ongoing request.
	// It can be raised if a file is not accessible or corrupt, for example.
	InternalError ErrorType = "Internal Error"
	// ServerError defines a programmatic error threw by the server, such as
	// a request cancellation.
	// It's a type of error that should be expected and handled.
	ServerError ErrorType = "Server Error"
	// NetworkError defines an error that comes from the network and its nodes.
	NetworkError ErrorType = "Network Error"
	// UserError defines an error that originated from the user and that
	// requires its intervention to correct it.
	// It can be raised if a passphrase is invalid, for example.
	UserError ErrorType = "User Error"
)

// LogType defines the type of log that is sent to the user.
type LogType string

var (
	InfoLog    LogType = "Info"
	WarningLog LogType = "Warning"
	ErrorLog   LogType = "Error"
	SuccessLog LogType = "Success"
)

// SelectedWallet holds the result of the wallet selection from the user.
type SelectedWallet struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

// SessionAPI builds the wallet JSON-RPC API with specific methods that are
// intended to be publicly exposed to third-party applications in a
// non-trustable environment.
// Because of the nature of the environment from where these methods are called,
// no administration methods are exposed. We don't want malicious third-party
// applications to leverage administration capabilities that could expose to the
// user and compromise his wallets.
func SessionAPI(log *zap.Logger, walletStore WalletStore, pipeline Pipeline, nodeSelector NodeSelector) (*jsonrpc.API, error) {
	sessions := NewSessions()

	walletAPI := jsonrpc.New(log)
	walletAPI.RegisterMethod("session.connect_wallet", NewConnectWallet(walletStore, pipeline, sessions))
	walletAPI.RegisterMethod("session.disconnect_wallet", NewDisconnectWallet(sessions))
	walletAPI.RegisterMethod("session.get_chain_id", NewGetChainID(nodeSelector))
	walletAPI.RegisterMethod("session.get_permissions", NewGetPermissions(sessions))
	walletAPI.RegisterMethod("session.list_keys", NewListKeys(sessions))
	walletAPI.RegisterMethod("session.request_permissions", NewRequestPermissions(walletStore, pipeline, sessions))
	walletAPI.RegisterMethod("session.sign_transaction", NewSignTransaction(pipeline, nodeSelector, sessions))
	walletAPI.RegisterMethod("session.send_transaction", NewSendTransaction(pipeline, nodeSelector, sessions))

	log.Info("the restricted JSON-RPC API has been initialised")

	return walletAPI, nil
}

// AdminAPI builds the JSON-RPC API of the wallet with all the methods available.
// This API exposes highly-sensitive methods, and, as a result, it should be
// only exposed to highly-trustable applications.
func AdminAPI(log *zap.Logger, walletStore WalletStore, netStore NetworkStore) (*jsonrpc.API, error) {
	walletAPI := jsonrpc.New(log)
	walletAPI.RegisterMethod("admin.annotate_key", NewAdminAnnotateKey(walletStore))
	walletAPI.RegisterMethod("admin.create_wallet", NewAdminCreateWallet(walletStore))
	walletAPI.RegisterMethod("admin.describe_key", NewAdminDescribeKey(walletStore))
	walletAPI.RegisterMethod("admin.describe_network", NewAdminDescribeNetwork(netStore))
	walletAPI.RegisterMethod("admin.describe_permissions", NewAdminDescribePermissions(walletStore))
	walletAPI.RegisterMethod("admin.describe_wallet", NewAdminDescribeWallet(walletStore))
	walletAPI.RegisterMethod("admin.generate_key", NewAdminGenerateKey(walletStore))
	walletAPI.RegisterMethod("admin.import_network", NewAdminImportNetwork(netStore))
	walletAPI.RegisterMethod("admin.import_wallet", NewAdminImportWallet(walletStore))
	walletAPI.RegisterMethod("admin.isolate_key", NewAdminIsolateKey(walletStore))
	walletAPI.RegisterMethod("admin.list_keys", NewAdminListKeys(walletStore))
	walletAPI.RegisterMethod("admin.list_networks", NewAdminListNetworks(netStore))
	walletAPI.RegisterMethod("admin.list_permissions", NewAdminListPermissions(walletStore))
	walletAPI.RegisterMethod("admin.list_wallets", NewAdminListWallets(walletStore))
	walletAPI.RegisterMethod("admin.purge_permissions", NewAdminPurgePermissions(walletStore))
	walletAPI.RegisterMethod("admin.remove_network", NewAdminRemoveNetwork(netStore))
	walletAPI.RegisterMethod("admin.remove_wallet", NewAdminRemoveWallet(walletStore))
	walletAPI.RegisterMethod("admin.revoke_permissions", NewAdminRevokePermissions(walletStore))
	walletAPI.RegisterMethod("admin.rotate_key", NewAdminRotateKey(walletStore))
	walletAPI.RegisterMethod("admin.send_message", &UnimplementedMethod{})
	walletAPI.RegisterMethod("admin.send_transaction", &UnimplementedMethod{})
	walletAPI.RegisterMethod("admin.sign_message", &UnimplementedMethod{})
	walletAPI.RegisterMethod("admin.sign_transaction", &UnimplementedMethod{})
	walletAPI.RegisterMethod("admin.taint_key", NewAdminTaintKey(walletStore))
	walletAPI.RegisterMethod("admin.untaint_key", NewAdminUntaintKey(walletStore))
	walletAPI.RegisterMethod("admin.update_permissions", NewAdminUpdatePermissions(walletStore))

	log.Info("the admin JSON-RPC API has been initialised")

	return walletAPI, nil
}

type UnimplementedMethod struct{}

func (u UnimplementedMethod) Handle(_ context.Context, _ jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	return nil, internalError(errors.New("this method is not implemented yet"))
}
