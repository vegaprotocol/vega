package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	coreversion "code.vegaprotocol.io/vega/version"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
)

const (
	// Network error codes are errors that comes from the network itself and its
	// nodes. It ranges from 1000 to 1999, included.
	// Apart from the communication failure, network errors are valued based on
	// their ABCI code counter-part:
	//     Network_Error_Code == ABCI_Error_Code + 1000

	// ErrorCodeNodeCommunicationFailed refers to the inability of the program to
	// talk to the network nodes.
	ErrorCodeNodeCommunicationFailed jsonrpc.ErrorCode = 1000

	// ErrorCodeNetworkRejectedTransaction refers to a transaction rejected by
	// the network nodes but for an unknown ABCI code.
	ErrorCodeNetworkRejectedTransaction jsonrpc.ErrorCode = 1001

	// ErrorCodeNetworkRejectedInvalidTransaction refers to a validation failure raised
	// by the network nodes (error code 51).
	ErrorCodeNetworkRejectedInvalidTransaction jsonrpc.ErrorCode = 1051

	// ErrorCodeNetworkRejectedMalformedTransaction refers to the inability to
	// decode a transaction from the network nodes (error code 60).
	ErrorCodeNetworkRejectedMalformedTransaction jsonrpc.ErrorCode = 1060

	// ErrorCodeNetworkCouldNotProcessTransaction refers to the inability to
	// process a transaction from the network nodes (error code 70).
	ErrorCodeNetworkCouldNotProcessTransaction jsonrpc.ErrorCode = 1070

	// ErrorCodeNetworkRejectedUnsupportedTransaction is raised when the network
	// nodes encounter an unsupported transaction (error code 80).
	ErrorCodeNetworkRejectedUnsupportedTransaction jsonrpc.ErrorCode = 1080

	// ErrorCodeNetworkSpamProtectionActivated is raised when the network
	// nodes spin up the spam protection mechanism (error code 89).
	ErrorCodeNetworkSpamProtectionActivated jsonrpc.ErrorCode = 1089

	// Application error codes are programmatic errors that comes from the API
	// itself and its "business" rules. It ranges from 2000 to 2999, included.

	// ErrorCodeRequestNotPermitted refers a request made by a third-party application
	// that is not permitted to do. This error is related to the permissions'
	// system.
	ErrorCodeRequestNotPermitted jsonrpc.ErrorCode = 2000

	// ErrorCodeRequestHasBeenCanceledByApplication refers to an automated cancellation of a
	// request by the application core. This happens when some requirements are
	// missing to ensure correct handling of a request.
	ErrorCodeRequestHasBeenCanceledByApplication jsonrpc.ErrorCode = 2001

	// ErrorCodeIncompatibilityBetweenNetworkAndSoftware refers to a
	// software that relies on a specific version of the network but the
	// network it tried to connect to is not the one expected.
	ErrorCodeIncompatibilityBetweenNetworkAndSoftware jsonrpc.ErrorCode = 2002

	// ErrorCodeServicePortAlreadyBound refers to a service that attempt to run
	// on a port that is already bound. The user should try to shut down the
	// software using that port, or update the configuration to use another port.
	ErrorCodeServicePortAlreadyBound jsonrpc.ErrorCode = 2003

	// User error codes are errors that results from the user. It ranges
	// from 3000 to 3999, included.

	// ErrorCodeConnectionHasBeenClosed refers to an interruption of the service triggered
	// by the user.
	ErrorCodeConnectionHasBeenClosed jsonrpc.ErrorCode = 3000

	// ErrorCodeRequestHasBeenRejected refers to an explicit rejection of a request by the
	// user. When received, the third-party application should consider the user
	// has withdrawn from the action, and thus, abort the action.
	ErrorCodeRequestHasBeenRejected jsonrpc.ErrorCode = 3001

	// ErrorCodeRequestHasBeenCanceledByUser refers to a cancellation of a request by the
	// user. It's conceptually different from a rejection. Contrary to a rejection,
	// when a cancellation is received, the third-party application should temporarily
	// back off, maintain its state, and wait for the user to be ready to continue.
	ErrorCodeRequestHasBeenCanceledByUser jsonrpc.ErrorCode = 3002
)

var (
	ErrAdminEndpointsNotExposed                           = errors.New("administrative endpoints are not exposed, for security reasons")
	ErrApplicationCanceledTheRequest                      = errors.New("the application canceled the request")
	ErrBlockHashIsRequired                                = errors.New("the block hash is required")
	ErrBlockHeightIsRequired                              = errors.New("the block-height is required")
	ErrCannotRotateKeysOnIsolatedWallet                   = errors.New("cannot rotate keys on an isolated wallet")
	ErrChainIDIsRequired                                  = errors.New("the chain ID is required")
	ErrConnectionTokenIsRequired                          = errors.New("the connection token is required")
	ErrCouldNotConnectToWallet                            = errors.New("could not connect to the wallet")
	ErrCouldNotGetChainIDFromNode                         = errors.New("could not get the chain ID from the node")
	ErrCouldNotGetLastBlockInformation                    = errors.New("could not get information about the last block on the network")
	ErrCouldNotRequestPermissions                         = errors.New("could not request permissions")
	ErrCouldNotSendTransaction                            = errors.New("could not send transaction")
	ErrCouldNotSignTransaction                            = errors.New("could not sign transaction")
	ErrCurrentPublicKeyDoesNotExist                       = errors.New("the current public key does not exist")
	ErrCurrentPublicKeyIsRequired                         = errors.New("the next public key is required")
	ErrEnactmentBlockHeightIsRequired                     = errors.New("the enactment block height is required")
	ErrEnactmentBlockHeightMustBeGreaterThanSubmissionOne = errors.New("the enactment block height must be greater than the submission one")
	ErrEncodedMessageIsNotValidBase64String               = errors.New("the encoded message is not a valid base-64 string")
	ErrEncodedSignatureIsNotValidBase64String             = errors.New("the encoded signature is not a valid base-64 string")
	ErrEncodedTransactionIsNotValidBase64String           = errors.New("the encoded transaction is not a valid base-64 string")
	ErrEncodedTransactionIsRequired                       = errors.New("the encoded transaction is required")
	ErrTransactionIsRequired                              = errors.New("the transaction or encoded transaction is required")
	ErrEncodedTransactionAndTransactionSupplied           = errors.New("both transaction and encodedTransaction supplied")
	ErrEncodedTransactionIsNotValid                       = errors.New("the encoded transaction is not valid")
	ErrHostnameIsRequired                                 = errors.New("the hostname is required")
	ErrInvalidLogLevelValue                               = errors.New("invalid log level value")
	ErrInvalidTokenExpiryValue                            = errors.New("invalid token expiry value")
	ErrIsolatedWalletPassphraseIsRequired                 = errors.New("the isolated wallet passphrase is required")
	ErrLastBlockDataOrNetworkIsRequired                   = errors.New("a network or the last block data is required")
	ErrMessageIsRequired                                  = errors.New("the message is required")
	ErrMethodWithoutParameters                            = errors.New("this method does not take any parameters")
	ErrMultipleNetworkSources                             = errors.New("network sources are mutually exclusive")
	ErrNetworkAlreadyExists                               = errors.New("a network with the same name already exists")
	ErrNetworkConfigurationDoesNotHaveGRPCNodes           = errors.New("the network does not have gRPC hosts configured")
	ErrNetworkCouldNotProcessTransaction                  = errors.New("the network could not process the transaction")
	ErrNetworkDoesNotExist                                = errors.New("the network does not exist")
	ErrNetworkIsRequired                                  = errors.New("the network is required")
	ErrNetworkNameIsRequired                              = errors.New("the network name is required")
	ErrNetworkOrNodeAddressIsRequired                     = errors.New("a network or a node address is required")
	ErrNetworkRejectedInvalidTransaction                  = errors.New("the network rejected the transaction because it's invalid")
	ErrNetworkRejectedMalformedTransaction                = errors.New("the network rejected the transaction because it's malformed")
	ErrNetworkRejectedUnsupportedTransaction              = errors.New("the network does not support this transaction")
	ErrNetworkSourceIsRequired                            = errors.New("a network source is required")
	ErrNetworkSpamProtectionActivated                     = errors.New("the network blocked the transaction through the spam protection")
	ErrNewNameIsRequired                                  = errors.New("the new name is required")
	ErrNewPassphraseIsRequired                            = errors.New("the new passphrase is required")
	ErrNextAndCurrentPublicKeysCannotBeTheSame            = errors.New("the next and current public keys cannot be the same")
	ErrNextPublicKeyDoesNotExist                          = errors.New("the next public key does not exist")
	ErrNextPublicKeyIsRequired                            = errors.New("the next public key is required")
	ErrNextPublicKeyIsTainted                             = errors.New("the next public key is tainted")
	ErrNoHealthyNodeAvailable                             = errors.New("no healthy node available")
	ErrNoWalletToConnectTo                                = errors.New("there is no wallet to connect to, you should, first, create or import a wallet")
	ErrParamsDoNotMatch                                   = errors.New("the params do not match expected ones")
	ErrParamsRequired                                     = errors.New("the params are required")
	ErrPassphraseIsRequired                               = errors.New("the passphrase is required")
	ErrProofOfWorkDifficultyRequired                      = errors.New("the proof-of-work difficulty is required")
	ErrProofOfWorkHashFunctionRequired                    = errors.New("the proof-of-work hash function is required")
	ErrPublicKeyDoesNotExist                              = errors.New("the public key does not exist")
	ErrPublicKeyIsNotAllowedToBeUsed                      = errors.New("the public key is not allowed to be used")
	ErrPublicKeyIsRequired                                = errors.New("the public key is required")
	ErrReadAccessOnPublicKeysRequired                     = errors.New(`a "read" access on public keys is required`)
	ErrRecoveryPhraseIsRequired                           = errors.New("the recovery phrase is required")
	ErrRequestInterrupted                                 = errors.New("the request has been interrupted")
	ErrRequestedPermissionsAreRequired                    = errors.New("the requested permissions are required")
	ErrSendingModeCannotBeTypeUnspecified                 = errors.New(`the sending mode can't be "TYPE_UNSPECIFIED"`)
	ErrSendingModeIsRequired                              = errors.New("the sending mode is required")
	ErrSignatureIsRequired                                = errors.New("the the signature is required")
	ErrSpecifyingNetworkAndLastBlockDataIsNotSupported    = errors.New("specifying a network and the last block data is not supported")
	ErrSpecifyingNetworkAndNodeAddressIsNotSupported      = errors.New("specifying a network and a node address is not supported")
	ErrSubmissionBlockHeightIsRequired                    = errors.New("the submission block height is required")
	ErrTokenDoesNotExist                                  = errors.New("the token does not exist")
	ErrTokenIsRequired                                    = errors.New("the token is required")
	ErrTransactionFailed                                  = errors.New("the transaction failed")
	ErrTransactionIsMalformed                             = errors.New("the transaction is malformed")
	ErrUserCanceledTheRequest                             = errors.New("the user canceled the request")
	ErrUserCloseTheConnection                             = errors.New("the user closed the connection")
	ErrUserRejectedTheRequest                             = errors.New("the user rejected the request")
	ErrWalletAlreadyExists                                = errors.New("a wallet with the same name already exists")
	ErrWalletDoesNotExist                                 = errors.New("the wallet does not exist")
	ErrWalletIsRequired                                   = errors.New("the wallet is required")
	ErrWalletNameIsRequired                               = errors.New("the wallet name is required")
	ErrWalletPassphraseIsRequired                         = errors.New("the wallet passphrase is required")
	ErrWalletKeyDerivationVersionIsRequired               = errors.New("the wallet key derivation version is required")
	ErrWrongPassphrase                                    = errors.New("wrong passphrase")
	ErrAPITokenExpirationCannotBeInThePast                = errors.New("the token expiration date cannot be set to a past date")
)

func applicationError(code jsonrpc.ErrorCode, err error) *jsonrpc.ErrorDetails {
	if code <= -32000 {
		panic("application error code should be greater than -32000")
	}
	return jsonrpc.NewCustomError(code, "Application error", err)
}

func userError(code jsonrpc.ErrorCode, err error) *jsonrpc.ErrorDetails {
	if code <= -32000 {
		panic("user error code should be greater than -32000")
	}
	return jsonrpc.NewCustomError(code, "User error", err)
}

func networkError(code jsonrpc.ErrorCode, err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewCustomError(code, "Network error", err)
}

// networkErrorFromTransactionError returns an error with a generic message but
// a specialized code. This is intended to give a coarse-grained indication
// to the third-party application without taking any risk of leaking information
// from the error message.
func networkErrorFromTransactionError(err error) *jsonrpc.ErrorDetails {
	txErr := types.TransactionError{}
	isTxErr := errors.As(err, &txErr)
	if !isTxErr {
		return networkError(ErrorCodeNodeCommunicationFailed, ErrTransactionFailed)
	}

	switch txErr.ABCICode {
	case 51:
		return networkError(ErrorCodeNetworkRejectedInvalidTransaction, ErrNetworkRejectedInvalidTransaction)
	case 60:
		return networkError(ErrorCodeNetworkRejectedMalformedTransaction, ErrNetworkRejectedMalformedTransaction)
	case 70:
		return networkError(ErrorCodeNetworkCouldNotProcessTransaction, ErrNetworkCouldNotProcessTransaction)
	case 80:
		return networkError(ErrorCodeNetworkRejectedUnsupportedTransaction, ErrNetworkRejectedUnsupportedTransaction)
	case 89:
		return networkError(ErrorCodeNetworkSpamProtectionActivated, ErrNetworkSpamProtectionActivated)
	default:
		return networkError(ErrorCodeNetworkRejectedTransaction, ErrTransactionFailed)
	}
}

func nodeCommunicationError(err error) *jsonrpc.ErrorDetails {
	return networkError(ErrorCodeNodeCommunicationFailed, err)
}

func invalidParams(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewInvalidParams(err)
}

func requestNotPermittedError(err error) *jsonrpc.ErrorDetails {
	return applicationError(ErrorCodeRequestNotPermitted, err)
}

func connectionClosedError(err error) *jsonrpc.ErrorDetails {
	return userError(ErrorCodeConnectionHasBeenClosed, err)
}

func requestInterruptedError(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewServerError(jsonrpc.ErrorCodeRequestHasBeenInterrupted, err)
}

func userCancellationError(err error) *jsonrpc.ErrorDetails {
	return userError(ErrorCodeRequestHasBeenCanceledByUser, err)
}

func userRejectionError() *jsonrpc.ErrorDetails {
	return userError(ErrorCodeRequestHasBeenRejected, ErrUserRejectedTheRequest)
}

func applicationCancellationError(err error) *jsonrpc.ErrorDetails {
	return applicationError(ErrorCodeRequestHasBeenCanceledByApplication, err)
}

func incompatibilityBetweenSoftwareAndNetworkError(networkVersion string) *jsonrpc.ErrorDetails {
	return applicationError(ErrorCodeIncompatibilityBetweenNetworkAndSoftware, fmt.Errorf("this software is not compatible with this network as the network is running version %s but this software expects the version %s", networkVersion, coreversion.Get()))
}

func servicePortAlreadyBound(err error) *jsonrpc.ErrorDetails {
	return applicationError(ErrorCodeServicePortAlreadyBound, err)
}

func internalError(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewInternalError(err)
}

// handleRequestFlowError is a generic function that build the appropriate
// API error response based on the underlying error.
// If none of them matches, the error handling is delegating to the caller.
func handleRequestFlowError(ctx context.Context, traceID string, interactor Interactor, err error) *jsonrpc.ErrorDetails {
	if errors.Is(err, ErrUserCloseTheConnection) {
		// This error means the user closed the connection by stopping the
		// wallet front-end application. As a result, there is no notification
		// to be sent to the user.
		return connectionClosedError(err)
	}
	if errors.Is(err, ErrRequestInterrupted) {
		interactor.NotifyError(ctx, traceID, ServerError, err)
		return requestInterruptedError(err)
	}
	if errors.Is(err, ErrUserCanceledTheRequest) {
		return userCancellationError(err)
	}
	return nil
}
