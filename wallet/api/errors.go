package api

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
)

const (
	// Network error codes are errors that comes from the network itself and its
	// nodes.

	// ErrorCodeNodeRequestFailed refers to the inability of the program to
	// talk to the network nodes.
	ErrorCodeNodeRequestFailed jsonrpc.ErrorCode = 1000

	// Application error codes are programmatic errors that comes from the API
	// itself and its "business" rules. It ranges from 1000 to 1999, included.

	// ErrorCodeRequestNotPermitted refers a request made by a third-party application
	// that is not permitted to do.
	ErrorCodeRequestNotPermitted jsonrpc.ErrorCode = 2000

	// User error codes are errors that results from the user. It ranges
	// from 2000 to 2999, included.

	// ErrorCodeConnectionHasBeenClosed refers to an interruption of the service triggered
	// by the user.
	ErrorCodeConnectionHasBeenClosed jsonrpc.ErrorCode = 3000

	// ErrorCodeRequestHasBeenRejected refers to an explicit rejection of a request by the
	// user.
	ErrorCodeRequestHasBeenRejected jsonrpc.ErrorCode = 3001
)

var (
	ErrCannotRotateKeysOnIsolatedWallet                   = errors.New("cannot rotate keys on an isolated wallet")
	ErrChainIDIsRequired                                  = errors.New("the chain ID is required")
	ErrUserRejectedTheRequest                             = errors.New("the user rejected the request")
	ErrUserCloseTheConnection                             = errors.New("the user closed the connection")
	ErrConnectionTokenIsRequired                          = errors.New("the connection token is required")
	ErrCouldNotConnectToWallet                            = errors.New("could not connect to the wallet")
	ErrCouldNotGetLastBlockInformation                    = errors.New("could not get information about the last block on the network")
	ErrCouldNotRequestPermissions                         = errors.New("could not request permissions")
	ErrCouldNotSendTransaction                            = errors.New("could not send transaction")
	ErrCouldNotSignTransaction                            = errors.New("could not sign transaction")
	ErrCurrentPublicKeyDoesNotExist                       = errors.New("the current public key does not exist")
	ErrCurrentPublicKeyIsRequired                         = errors.New("the next public key is required")
	ErrEnactmentBlockHeightIsRequired                     = errors.New("the enactment block height is required")
	ErrEnactmentBlockHeightMustBeGreaterThanSubmissionOne = errors.New("the enactment block height must be greater than the submission one")
	ErrEncodedTransactionIsNotValidBase64String           = errors.New("the encoded transaction is not a valid base-64 string")
	ErrEncodedTransactionIsRequired                       = errors.New("the encoded transaction is required")
	ErrHostnameIsRequired                                 = errors.New("the hostname is required")
	ErrNextAndCurrentPublicKeysCannotBeTheSame            = errors.New("the next and current public keys cannot be the same")
	ErrNextPublicKeyDoesNotExist                          = errors.New("the next public key does not exist")
	ErrNextPublicKeyIsRequired                            = errors.New("the next public key is required")
	ErrNextPublicKeyIsTainted                             = errors.New("the next public key is tainted")
	ErrNoHealthyNodeAvailable                             = errors.New("no healthy node available")
	ErrParamsDoNotMatch                                   = errors.New("the params do not match expected ones")
	ErrParamsRequired                                     = errors.New("the params are required")
	ErrPassphraseIsRequired                               = errors.New("the passphrase is required")
	ErrPublicKeyDoesNotExist                              = errors.New("the public key does not exist")
	ErrPublicKeyIsNotAllowedToBeUsed                      = errors.New("the public key is not allowed to be used")
	ErrPublicKeyIsRequired                                = errors.New("the public key is required")
	ErrReadAccessOnPublicKeysRequired                     = errors.New(`a "read" access on public keys is required`)
	ErrRecoveryPhraseIsRequired                           = errors.New("the recovery phrase is required")
	ErrRequestInterrupted                                 = errors.New("the request has been interrupted")
	ErrRequestedPermissionsAreRequired                    = errors.New("the requested permissions are required")
	ErrSendingModeCannotBeTypeUnspecified                 = errors.New(`the sending mode can't be "TYPE_UNSPECIFIED"`)
	ErrSendingModeIsRequired                              = errors.New("the sending mode is required")
	ErrSubmissionBlockHeightIsRequired                    = errors.New("the submission block height is required")
	ErrTransactionFailed                                  = errors.New("the transaction failed")
	ErrWalletAlreadyExists                                = errors.New("a wallet with the same name already exists")
	ErrWalletDoesNotExist                                 = errors.New("the wallet does not exist")
	ErrWalletIsRequired                                   = errors.New("the wallet is required")
	ErrWalletVersionIsRequired                            = errors.New("the wallet version is required")
	ErrNetworkDoesNotExist                                = errors.New("the network does not exist")
	ErrNetworkIsRequired                                  = errors.New("the network is required")
	ErrNetworkSourceIsRequired                            = errors.New("a network source is required")
	ErrMultipleNetworkSources                             = errors.New("network sources are mutually exclusive")
	ErrNetworkAlreadyExists                               = errors.New("a network with the same name already exists")
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
	if code <= -32000 {
		panic("network error code should be greater than -32000")
	}
	return jsonrpc.NewCustomError(code, "Network error", err)
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

func userRejectionError() *jsonrpc.ErrorDetails {
	return userError(ErrorCodeRequestHasBeenRejected, ErrUserRejectedTheRequest)
}

func internalError(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewInternalError(err)
}

// handleRequestFlowError is a generic function that build the appropriate
// API error response based on the underlying error.
// If none of them matches, the error handling is delegating to the caller.
func handleRequestFlowError(ctx context.Context, traceID string, pipeline Pipeline, err error) *jsonrpc.ErrorDetails {
	if errors.Is(err, ErrUserCloseTheConnection) {
		// This error means the user closed the connection by stopping the
		// wallet front-end application. As a result, there is no notification
		// to be sent to the user.
		return connectionClosedError(err)
	}
	if errors.Is(err, ErrRequestInterrupted) {
		pipeline.NotifyError(ctx, traceID, ServerError, err)
		return requestInterruptedError(err)
	}
	return nil
}
