package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/api/node"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"go.uber.org/zap"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/wallet/api WalletStore,NetworkStore,Interactor,ConnectionsManager,SpamHandler

type NodeSelectorBuilder func(hosts []string, retries uint64) (node.Selector, error)

// WalletStore is the component used to retrieve and update wallets from the
// computer.
type WalletStore interface {
	UnlockWallet(ctx context.Context, name, passphrase string) error
	LockWallet(ctx context.Context, name string) error
	WalletExists(ctx context.Context, name string) (bool, error)
	GetWallet(ctx context.Context, name string) (wallet.Wallet, error)
	ListWallets(ctx context.Context) ([]string, error)
	CreateWallet(ctx context.Context, w wallet.Wallet, passphrase string) error
	UpdateWallet(ctx context.Context, w wallet.Wallet) error
	UpdatePassphrase(ctx context.Context, name, newPassphrase string) error
	DeleteWallet(ctx context.Context, name string) error
	RenameWallet(ctx context.Context, currentName, newName string) error
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

type ConnectionsManager interface {
	EndSessionConnection(hostname, wallet string)
	ListSessionConnections() []Connection
}

type SpamHandler interface {
	GenerateProofOfWork(pubKey string, blockData *nodetypes.SpamStatistics) (*commandspb.ProofOfWork, error)
	CheckSubmission(req *walletpb.SubmitTransactionRequest, latest *nodetypes.SpamStatistics) error
}

// Interactor is the component in charge of delegating the JSON-RPC API
// requests, notifications and logs to the wallet front-end.
// Convention:
//   - Notify* calls do not expect a response from the user.
//   - Request* calls are expecting a response from the user.
//   - Log function is just information logging and does not expect a response.
//
//nolint:interfacebloat
type Interactor interface {
	// NotifyInteractionSessionBegan notifies the beginning of an interaction
	// session.
	// A session is scoped to a request.
	NotifyInteractionSessionBegan(ctx context.Context, traceID string) error

	// NotifyInteractionSessionEnded notifies the end of an interaction
	// session.
	// A session is scoped to a request.
	NotifyInteractionSessionEnded(ctx context.Context, traceID string)

	// NotifySuccessfulTransaction is used to report a successful transaction.
	NotifySuccessfulTransaction(ctx context.Context, traceID, txHash, deserializedInputData, tx string, sentAt time.Time, host string)

	// NotifyFailedTransaction is used to report a failed transaction.
	NotifyFailedTransaction(ctx context.Context, traceID, deserializedInputData, tx string, err error, sentAt time.Time, host string)

	// NotifySuccessfulRequest is used to notify the user the request is
	// successful.
	NotifySuccessfulRequest(ctx context.Context, traceID string, message string)

	// NotifyError is used to report errors to the user.
	NotifyError(ctx context.Context, traceID string, t ErrorType, err error)

	// Log is used to report information of any kind to the user. This is used
	// to log internal activities and provide feedback to the wallet front-ends.
	// It's purely for informational purpose.
	// Receiving logs should be expected at any point during an interaction
	// session.
	Log(ctx context.Context, traceID string, t LogType, msg string)

	// RequestWalletConnectionReview is used to trigger a user review of
	// the wallet connection requested by the specified hostname.
	// It returns the type of connection approval chosen by the user.
	RequestWalletConnectionReview(ctx context.Context, traceID, hostname string) (string, error)

	// RequestWalletSelection is used to trigger the selection of the wallet the
	// user wants to use for the specified hostname.
	RequestWalletSelection(ctx context.Context, traceID, hostname string, availableWallets []string) (SelectedWallet, error)

	// RequestPassphrase is used to request to the user the passphrase of a wallet.
	// It's primarily used by requests that update the wallet.
	RequestPassphrase(ctx context.Context, traceID, wallet string) (string, error)

	// RequestPermissionsReview is used to trigger a user review of the permissions
	// requested by the specified hostname.
	// It returns true if the user approved the requested permissions, false
	// otherwise.
	RequestPermissionsReview(ctx context.Context, traceID, hostname, wallet string, perms map[string]string) (bool, error)

	// RequestTransactionReviewForSending is used to trigger a user review of the
	// transaction a third-party application wants to send.
	// It returns true if the user approved the sending of the transaction,
	// false otherwise.
	RequestTransactionReviewForSending(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error)

	// RequestTransactionReviewForSigning is used to trigger a user review of the
	// transaction a third-party application wants to sign. The wallet doesn't
	// send the transaction.
	// It returns true if the user approved the signing of the transaction,
	// false otherwise.
	RequestTransactionReviewForSigning(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error)
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
	InternalError ErrorType = "Internal error"
	// ServerError defines a programmatic error threw by the server, such as
	// a request cancellation. The server error targets error that happens in
	// the communication layer. It's different form application error.
	// It's a type of error that should be expected and handled.
	ServerError ErrorType = "Server error"
	// NetworkError defines an error that comes from the network and its nodes.
	NetworkError ErrorType = "Network error"
	// ApplicationError defines a programmatic error threw by the application
	// core, also called "business logic".
	ApplicationError ErrorType = "Application error"
	// UserError defines an error that originated from the user and that
	// requires its intervention to correct it.
	// It can be raised if a passphrase is invalid, for example.
	UserError ErrorType = "User error"
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

type ClientAPI struct {
	connectWallet   *ClientConnectWallet
	getChainID      *ClientGetChainID
	listKeys        *ClientListKeys
	signTransaction *ClientSignTransaction
	sendTransaction *ClientSendTransaction
}

func (a *ClientAPI) ConnectWallet(ctx context.Context, hostname string) (wallet.Wallet, *jsonrpc.ErrorDetails) {
	return a.connectWallet.Handle(ctx, hostname)
}

func (a *ClientAPI) GetChainID(ctx context.Context) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	return a.getChainID.Handle(ctx)
}

func (a *ClientAPI) ListKeys(ctx context.Context, connectedWallet ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	return a.listKeys.Handle(ctx, connectedWallet)
}

func (a *ClientAPI) SignTransaction(ctx context.Context, rawParams jsonrpc.Params, connectedWallet ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	return a.signTransaction.Handle(ctx, rawParams, connectedWallet)
}

func (a *ClientAPI) SendTransaction(ctx context.Context, rawParams jsonrpc.Params, connectedWallet ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	return a.sendTransaction.Handle(ctx, rawParams, connectedWallet)
}

func BuildClientAPI(walletStore WalletStore, interactor Interactor, nodeSelector node.Selector, spam SpamHandler) (*ClientAPI, error) {
	clientAPI := &ClientAPI{}

	clientAPI.connectWallet = NewConnectWallet(walletStore, interactor)
	clientAPI.getChainID = NewGetChainID(nodeSelector)
	clientAPI.listKeys = NewListKeys(walletStore, interactor)
	clientAPI.signTransaction = NewClientSignTransaction(walletStore, interactor, nodeSelector, spam)
	clientAPI.sendTransaction = NewClientSendTransaction(walletStore, interactor, nodeSelector, spam)

	return clientAPI, nil
}

// AdminAPI builds the JSON-RPC API of the wallet with all the methods available.
// This API exposes highly-sensitive methods, and, as a result, it should be
// only exposed to highly-trustable applications.
func AdminAPI(
	log *zap.Logger,
	walletStore WalletStore,
	netStore NetworkStore,
	nodeSelectorBuilder NodeSelectorBuilder,
	connectionsManager ConnectionsManager,
) (*jsonrpc.Dispatcher, error) {
	walletAPI := jsonrpc.NewDispatcher(log)
	walletAPI.RegisterMethod("admin.annotate_key", NewAdminAnnotateKey(walletStore))
	walletAPI.RegisterMethod("admin.close_connection", NewAdminCloseConnection(connectionsManager))
	walletAPI.RegisterMethod("admin.close_connections_to_hostname", NewAdminCloseConnectionsToHostname(connectionsManager))
	walletAPI.RegisterMethod("admin.close_connections_to_wallet", NewAdminCloseConnectionsToWallet(connectionsManager))
	walletAPI.RegisterMethod("admin.create_wallet", NewAdminCreateWallet(walletStore))
	walletAPI.RegisterMethod("admin.describe_key", NewAdminDescribeKey(walletStore))
	walletAPI.RegisterMethod("admin.describe_network", NewAdminDescribeNetwork(netStore))
	walletAPI.RegisterMethod("admin.describe_permissions", NewAdminDescribePermissions(walletStore))
	walletAPI.RegisterMethod("admin.describe_wallet", NewAdminDescribeWallet(walletStore))
	walletAPI.RegisterMethod("admin.generate_key", NewAdminGenerateKey(walletStore))
	walletAPI.RegisterMethod("admin.import_network", NewAdminImportNetwork(netStore))
	walletAPI.RegisterMethod("admin.import_wallet", NewAdminImportWallet(walletStore))
	walletAPI.RegisterMethod("admin.isolate_key", NewAdminIsolateKey(walletStore))
	walletAPI.RegisterMethod("admin.list_connections", NewAdminListConnections(connectionsManager))
	walletAPI.RegisterMethod("admin.list_keys", NewAdminListKeys(walletStore))
	walletAPI.RegisterMethod("admin.list_networks", NewAdminListNetworks(netStore))
	walletAPI.RegisterMethod("admin.list_permissions", NewAdminListPermissions(walletStore))
	walletAPI.RegisterMethod("admin.list_wallets", NewAdminListWallets(walletStore))
	walletAPI.RegisterMethod("admin.purge_permissions", NewAdminPurgePermissions(walletStore))
	walletAPI.RegisterMethod("admin.remove_network", NewAdminRemoveNetwork(netStore))
	walletAPI.RegisterMethod("admin.remove_wallet", NewAdminRemoveWallet(walletStore))
	walletAPI.RegisterMethod("admin.rename_wallet", NewAdminRenameWallet(walletStore))
	walletAPI.RegisterMethod("admin.revoke_permissions", NewAdminRevokePermissions(walletStore))
	walletAPI.RegisterMethod("admin.rotate_key", NewAdminRotateKey(walletStore))
	walletAPI.RegisterMethod("admin.send_raw_transaction", NewAdminSendRawTransaction(netStore, nodeSelectorBuilder))
	walletAPI.RegisterMethod("admin.send_transaction", NewAdminSendTransaction(walletStore, netStore, nodeSelectorBuilder))
	walletAPI.RegisterMethod("admin.sign_message", NewAdminSignMessage(walletStore))
	walletAPI.RegisterMethod("admin.sign_transaction", NewAdminSignTransaction(walletStore, netStore, nodeSelectorBuilder))
	walletAPI.RegisterMethod("admin.taint_key", NewAdminTaintKey(walletStore))
	walletAPI.RegisterMethod("admin.untaint_key", NewAdminUntaintKey(walletStore))
	walletAPI.RegisterMethod("admin.update_network", NewAdminUpdateNetwork(netStore))
	walletAPI.RegisterMethod("admin.update_passphrase", NewAdminUpdatePassphrase(walletStore))
	walletAPI.RegisterMethod("admin.update_permissions", NewAdminUpdatePermissions(walletStore))
	walletAPI.RegisterMethod("admin.verify_message", NewAdminVerifyMessage())

	log.Info("the admin JSON-RPC API has been initialised")

	return walletAPI, nil
}

func noNodeSelectionReporting(_ node.ReportType, _ string) {
	// Nothing to do.
}
