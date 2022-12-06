package api

import (
	"context"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/service"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"go.uber.org/zap"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/wallet/api WalletStore,NetworkStore,Interactor,ServiceStore,TokenStore,TimeProvider

type NodeSelectorBuilder func(hosts []string, retries uint64) (node.Selector, error)

// WalletStore is the component used to retrieve and update wallets from the
// computer.
type WalletStore interface {
	WalletExists(ctx context.Context, name string) (bool, error)
	GetWallet(ctx context.Context, name, passphrase string) (wallet.Wallet, error)
	ListWallets(ctx context.Context) ([]string, error)
	SaveWallet(ctx context.Context, w wallet.Wallet, passphrase string) error
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

// ServiceStore is used to initialise the RSA keys the service API v1 relies on.
type ServiceStore interface {
	RSAKeysExists() (bool, error)
	SaveRSAKeys(*service.RSAKeys) error
	GetRsaKeys() (*service.RSAKeys, error)
}

// TokenStore is the component used to retrieve and update the API tokens from the
// computer.
type TokenStore interface {
	TokenExists(token string) (bool, error)
	ListTokens() ([]session.TokenSummary, error)
	GetToken(token string) (session.Token, error)
	SaveToken(tokenConfig session.Token) error
	DeleteToken(token string) error
}

// PolicyBuilderFunc return the policy the API v2.
type PolicyBuilderFunc func(ctx context.Context) service.Policy

// InteractorBuilderFunc returns the interactor to use in the client API.
type InteractorBuilderFunc func(ctx context.Context) Interactor

// ShutdownSwitchBuilder is used to build a switch that is controlled by
// components that share the same lifecycle.
type ShutdownSwitchBuilder func() *ServiceShutdownSwitch

// LoggerBuilderFunc is used to build a logger. It returns the built logger and a
// zap.AtomicLevel to allow the caller to dynamically change the log level.
type LoggerBuilderFunc func(path paths.StatePath, level string) (*zap.Logger, zap.AtomicLevel, error)

// EventListener is used to transmit event happening in a component to another one.
// For example, it's used in the service to broadcast information about the
// service health and state.
type EventListener func(eventName string, optionalData ...interface{})

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
	NotifySuccessfulTransaction(ctx context.Context, traceID, txHash, deserializedInputData, tx string, sentAt time.Time)

	// NotifyFailedTransaction is used to report a failed transaction.
	NotifyFailedTransaction(ctx context.Context, traceID, deserializedInputData, tx string, err error, sentAt time.Time)

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

// ClientAPI builds the wallet JSON-RPC API with specific methods that are
// intended to be publicly exposed to third-party applications in a
// non-trustable environment.
// Because of the nature of the environment from where these methods are called,
// no administration methods are exposed. We don't want malicious third-party
// applications to leverage administration capabilities that could expose the
// user and/or compromise his wallets.
func ClientAPI(log *zap.Logger, walletStore WalletStore, interactor Interactor, nodeSelector node.Selector, sessions *session.Sessions) (*jsonrpc.API, error) {
	walletAPI := jsonrpc.New(log)

	// We add this pre-check so users stop asking why they can't access the
	// administrative endpoints.
	walletAPI.AddDispatchPolicy(func(_ context.Context, request jsonrpc.Request, _ jsonrpc.RequestMetadata) *jsonrpc.ErrorDetails {
		if strings.HasPrefix(request.Method, "admin.") {
			return requestNotPermittedError(ErrAdminEndpointsNotExposed)
		}
		return nil
	})

	walletAPI.RegisterMethod("client.connect_wallet", NewConnectWallet(walletStore, interactor, sessions))
	walletAPI.RegisterMethod("client.disconnect_wallet", NewDisconnectWallet(sessions))
	walletAPI.RegisterMethod("client.get_chain_id", NewGetChainID(nodeSelector))
	walletAPI.RegisterMethod("client.list_keys", NewListKeys(walletStore, interactor, sessions))
	walletAPI.RegisterMethod("client.sign_transaction", NewSignTransaction(interactor, nodeSelector, sessions))
	walletAPI.RegisterMethod("client.send_transaction", NewSendTransaction(interactor, nodeSelector, sessions))
	walletAPI.RegisterMethod("client.request_permissions", NewRequestPermissions(walletStore, interactor, sessions))
	walletAPI.RegisterMethod("client.get_permissions", NewGetPermissions(sessions))

	log.Info("the client JSON-RPC API has been initialised")

	return walletAPI, nil
}

// AdminAPI builds the JSON-RPC API of the wallet with all the methods available.
// This API exposes highly-sensitive methods, and, as a result, it should be
// only exposed to highly-trustable applications.
func AdminAPI(
	log *zap.Logger,
	walletStore WalletStore,
	netStore NetworkStore,
	svcStore ServiceStore,
	tokenStore TokenStore,
	nodeSelectorBuilder NodeSelectorBuilder,
	policyBuilderFunc PolicyBuilderFunc,
	interactorBuilderFunc InteractorBuilderFunc,
	loggerBuilderFunc LoggerBuilderFunc,
	contextBuilderFunc ShutdownSwitchBuilder,
) (*jsonrpc.API, error) {
	servicesManager := NewServicesManager(tokenStore, walletStore)

	walletAPI := jsonrpc.New(log)
	walletAPI.RegisterMethod("admin.annotate_key", NewAdminAnnotateKey(walletStore))
	walletAPI.RegisterMethod("admin.close_connection", NewAdminCloseConnection(servicesManager))
	walletAPI.RegisterMethod("admin.close_connections_to_hostname", NewAdminCloseConnectionsToHostname(servicesManager))
	walletAPI.RegisterMethod("admin.close_connections_to_wallet", NewAdminCloseConnectionsToWallet(servicesManager))
	walletAPI.RegisterMethod("admin.create_wallet", NewAdminCreateWallet(walletStore))
	walletAPI.RegisterMethod("admin.delete_api_token", NewAdminDeleteAPIToken(tokenStore))
	walletAPI.RegisterMethod("admin.describe_key", NewAdminDescribeKey(walletStore))
	walletAPI.RegisterMethod("admin.describe_network", NewAdminDescribeNetwork(netStore))
	walletAPI.RegisterMethod("admin.describe_permissions", NewAdminDescribePermissions(walletStore))
	walletAPI.RegisterMethod("admin.describe_tokens", NewAdminDescribeAPIToken(tokenStore))
	walletAPI.RegisterMethod("admin.describe_wallet", NewAdminDescribeWallet(walletStore))
	walletAPI.RegisterMethod("admin.generate_api_token", NewAdminGenerateAPIToken(walletStore, tokenStore))
	walletAPI.RegisterMethod("admin.generate_key", NewAdminGenerateKey(walletStore))
	walletAPI.RegisterMethod("admin.import_network", NewAdminImportNetwork(netStore))
	walletAPI.RegisterMethod("admin.import_wallet", NewAdminImportWallet(walletStore))
	walletAPI.RegisterMethod("admin.isolate_key", NewAdminIsolateKey(walletStore))
	walletAPI.RegisterMethod("admin.list_connections", NewAdminListConnections(servicesManager))
	walletAPI.RegisterMethod("admin.list_keys", NewAdminListKeys(walletStore))
	walletAPI.RegisterMethod("admin.list_networks", NewAdminListNetworks(netStore))
	walletAPI.RegisterMethod("admin.list_permissions", NewAdminListPermissions(walletStore))
	walletAPI.RegisterMethod("admin.list_tokens", NewAdminListAPITokens(tokenStore))
	walletAPI.RegisterMethod("admin.list_wallets", NewAdminListWallets(walletStore))
	walletAPI.RegisterMethod("admin.purge_permissions", NewAdminPurgePermissions(walletStore))
	walletAPI.RegisterMethod("admin.remove_network", NewAdminRemoveNetwork(netStore))
	walletAPI.RegisterMethod("admin.remove_wallet", NewAdminRemoveWallet(walletStore))
	walletAPI.RegisterMethod("admin.rename_wallet", NewAdminRenameWallet(walletStore))
	walletAPI.RegisterMethod("admin.revoke_permissions", NewAdminRevokePermissions(walletStore))
	walletAPI.RegisterMethod("admin.rotate_key", NewAdminRotateKey(walletStore))
	walletAPI.RegisterMethod("admin.send_transaction", NewAdminSendTransaction(walletStore, netStore, nodeSelectorBuilder))
	walletAPI.RegisterMethod("admin.send_raw_transaction", NewAdminSendRawTransaction(netStore, nodeSelectorBuilder))
	walletAPI.RegisterMethod("admin.sign_message", NewAdminSignMessage(walletStore))
	walletAPI.RegisterMethod("admin.sign_transaction", NewAdminSignTransaction(walletStore, netStore, nodeSelectorBuilder))
	walletAPI.RegisterMethod("admin.start_service", NewAdminStartService(walletStore, netStore, svcStore, policyBuilderFunc, interactorBuilderFunc, loggerBuilderFunc, contextBuilderFunc, servicesManager))
	walletAPI.RegisterMethod("admin.stop_service", NewAdminStopService(servicesManager))
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
