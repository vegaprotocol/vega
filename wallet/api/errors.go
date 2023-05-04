package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
)

const (
	// Implementation-defined server-errors.

	// ErrorCodeRequestHasBeenInterrupted refers to a request that has been
	// interrupted by the server or the third-party application. It could
	// originate from a timeout or an explicit cancellation.
	ErrorCodeRequestHasBeenInterrupted jsonrpc.ErrorCode = -32001

	// ErrorCodeHostnameResolutionFailure refers to the inability for the server
	// to resolve the hostname from the request.
	ErrorCodeHostnameResolutionFailure jsonrpc.ErrorCode = -32002

	// ErrorCodeAuthenticationFailure refers to a request that have authentication
	// problems.
	ErrorCodeAuthenticationFailure jsonrpc.ErrorCode = -32003

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

	// ErrorCodeRequestHasBeenCancelledByApplication refers to an automated
	// cancellation of a request by the application core. This happens when some
	// requirements are missing to ensure correct handling of a request.
	ErrorCodeRequestHasBeenCancelledByApplication jsonrpc.ErrorCode = 2001

	// User error codes are errors that results from the user. It ranges
	// from 3000 to 3999, included.

	// ErrorCodeConnectionHasBeenClosed refers to an interruption of the service
	// triggered by the user.
	ErrorCodeConnectionHasBeenClosed jsonrpc.ErrorCode = 3000

	// ErrorCodeRequestHasBeenRejected refers to an explicit rejection of a
	// request by the user. When received, the third-party application should
	// consider the user has withdrawn from the action, and thus, abort the
	// action.
	ErrorCodeRequestHasBeenRejected jsonrpc.ErrorCode = 3001

	// ErrorCodeRequestHasBeenCancelledByUser refers to a cancellation of a
	// request by the user. It's conceptually different from a rejection.
	// Contrary to a rejection, when a cancellation is received, the third-party
	// application should temporarily back off, maintain its state, and wait for
	// the user to be ready to continue.
	ErrorCodeRequestHasBeenCancelledByUser jsonrpc.ErrorCode = 3002
)

var (
	ErrApplicationCancelledTheRequest                     = errors.New("the application cancelled the request")
	ErrBlockHashIsRequired                                = errors.New("the block hash is required")
	ErrBlockHeightIsRequired                              = errors.New("the block height is required")
	ErrBlockHeightTooHistoric                             = errors.New("the block height is too historic")
	ErrCannotRotateKeysOnIsolatedWallet                   = errors.New("cannot rotate keys on an isolated wallet")
	ErrChainIDIsRequired                                  = errors.New("the chain ID is required")
	ErrConnectionClosed                                   = errors.New("the connection has been closed")
	ErrCouldNotCheckTransaction                           = errors.New("could not check transaction")
	ErrCouldNotConnectToWallet                            = errors.New("could not connect to the wallet")
	ErrCouldNotGetChainIDFromNode                         = errors.New("could not get the chain ID from the node")
	ErrCouldNotGetLastBlockInformation                    = errors.New("could not get information about the last block on the network")
	ErrCouldNotListKeys                                   = errors.New("could not list the keys")
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
	ErrHostnameIsRequired                                 = errors.New("the hostname is required")
	ErrIsolatedWalletPassphraseIsRequired                 = errors.New("the isolated wallet passphrase is required")
	ErrLastBlockDataOrNetworkIsRequired                   = errors.New("a network or the last block data is required")
	ErrMessageIsRequired                                  = errors.New("the message is required")
	ErrNetworkAlreadyExists                               = errors.New("a network with the same name already exists")
	ErrNetworkConfigurationDoesNotHaveGRPCNodes           = errors.New("the network does not have gRPC hosts configured")
	ErrNetworkDoesNotExist                                = errors.New("the network does not exist")
	ErrNetworkIsRequired                                  = errors.New("the network is required")
	ErrNetworkNameIsRequired                              = errors.New("the network name is required")
	ErrNetworkOrNodeAddressIsRequired                     = errors.New("a network or a node address is required")
	ErrNetworkSourceIsRequired                            = errors.New("a network source is required")
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
	ErrPublicKeyIsNotAllowedToBeUsed                      = errors.New("this public key is not allowed to be used")
	ErrPublicKeyIsRequired                                = errors.New("the public key is required")
	ErrRawTransactionIsNotValidVegaTransaction            = errors.New("the raw transaction is not a valid Vega transaction")
	ErrRecoveryPhraseIsRequired                           = errors.New("the recovery phrase is required")
	ErrRequestCancelled                                   = errors.New("the request has been cancelled")
	ErrRequestInterrupted                                 = errors.New("the request has been interrupted")
	ErrSendingModeCannotBeTypeUnspecified                 = errors.New(`the sending mode can't be "TYPE_UNSPECIFIED"`)
	ErrSendingModeIsRequired                              = errors.New("the sending mode is required")
	ErrSignatureIsRequired                                = errors.New("the signature is required")
	ErrSpecifyingNetworkAndLastBlockDataIsNotSupported    = errors.New("specifying a network and the last block data is not supported")
	ErrSpecifyingNetworkAndNodeAddressIsNotSupported      = errors.New("specifying a network and a node address is not supported")
	ErrSubmissionBlockHeightIsRequired                    = errors.New("the submission block height is required")
	ErrTransactionIsNotValidJSON                          = errors.New("the transaction is not valid JSON")
	ErrTransactionIsRequired                              = errors.New("the transaction is required")
	ErrTransactionsPerBlockLimitReached                   = errors.New("the transaction per block limit has been reached")
	ErrUserCancelledTheRequest                            = errors.New("the user cancelled the request")
	ErrUserCloseTheConnection                             = errors.New("the user closed the connection")
	ErrUserRejectedAccessToKeys                           = errors.New("the user rejected the access to the keys")
	ErrUserRejectedCheckingOfTransaction                  = errors.New("the user rejected the checking of the transaction")
	ErrUserRejectedSendingOfTransaction                   = errors.New("the user rejected the sending of the transaction")
	ErrUserRejectedSigningOfTransaction                   = errors.New("the user rejected the signing of the transaction")
	ErrUserRejectedWalletConnection                       = errors.New("the user rejected the wallet connection")
	ErrWalletAlreadyExists                                = errors.New("a wallet with the same name already exists")
	ErrWalletDoesNotExist                                 = errors.New("the wallet does not exist")
	ErrWalletIsLocked                                     = errors.New("the wallet is locked")
	ErrWalletIsRequired                                   = errors.New("the wallet is required")
	ErrWalletKeyDerivationVersionIsRequired               = errors.New("the wallet key derivation version is required")
	ErrWrongPassphrase                                    = errors.New("wrong passphrase")
)

func ApplicationError(code jsonrpc.ErrorCode, err error) *jsonrpc.ErrorDetails {
	if code <= -32000 {
		panic("application error code should be greater than -32000")
	}
	return jsonrpc.NewCustomError(code, "Application error", err)
}

func UserError(code jsonrpc.ErrorCode, err error) *jsonrpc.ErrorDetails {
	if code <= -32000 {
		panic("user error code should be greater than -32000")
	}
	return jsonrpc.NewCustomError(code, "User error", err)
}

func NetworkError(code jsonrpc.ErrorCode, err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewCustomError(code, "Network error", err)
}

// NetworkErrorFromTransactionError returns an error with a generic message but
// a specialized code. This is intended to give a coarse-grained indication
// to the third-party application without taking any risk of leaking information
// from the error message.
func NetworkErrorFromTransactionError(err error) *jsonrpc.ErrorDetails {
	txErr := types.TransactionError{}
	isTxErr := errors.As(err, &txErr)
	if !isTxErr {
		return NetworkError(ErrorCodeNodeCommunicationFailed, fmt.Errorf("the transaction failed: %w", err))
	}

	switch txErr.ABCICode {
	case 51:
		return NetworkError(ErrorCodeNetworkRejectedInvalidTransaction, fmt.Errorf("the network rejected the transaction because it's invalid: %w", err))
	case 60:
		return NetworkError(ErrorCodeNetworkRejectedMalformedTransaction, fmt.Errorf("the network rejected the transaction because it's malformed: %w", err))
	case 70:
		return NetworkError(ErrorCodeNetworkCouldNotProcessTransaction, fmt.Errorf("the network could not process the transaction: %w", err))
	case 80:
		return NetworkError(ErrorCodeNetworkRejectedUnsupportedTransaction, fmt.Errorf("the network does not support this transaction: %w", err))
	case 89:
		return NetworkError(ErrorCodeNetworkSpamProtectionActivated, fmt.Errorf("the network blocked the transaction through the spam protection: %w", err))
	default:
		return NetworkError(ErrorCodeNetworkRejectedTransaction, fmt.Errorf("the transaction failed: %w", err))
	}
}

func NodeCommunicationError(err error) *jsonrpc.ErrorDetails {
	return NetworkError(ErrorCodeNodeCommunicationFailed, err)
}

func InvalidParams(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewInvalidParams(err)
}

func RequestInterruptedError(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewServerError(ErrorCodeRequestHasBeenInterrupted, err)
}

func ConnectionClosedError(err error) *jsonrpc.ErrorDetails {
	return UserError(ErrorCodeConnectionHasBeenClosed, err)
}

func UserCancellationError(err error) *jsonrpc.ErrorDetails {
	return UserError(ErrorCodeRequestHasBeenCancelledByUser, err)
}

func UserRejectionError(err error) *jsonrpc.ErrorDetails {
	return UserError(ErrorCodeRequestHasBeenRejected, err)
}

func RequestNotPermittedError(err error) *jsonrpc.ErrorDetails {
	return ApplicationError(ErrorCodeRequestNotPermitted, err)
}

func ApplicationCancellationError(err error) *jsonrpc.ErrorDetails {
	return ApplicationError(ErrorCodeRequestHasBeenCancelledByApplication, err)
}

func InternalError(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewInternalError(err)
}

// HandleRequestFlowError is a generic function that build the appropriate
// API error response based on the underlying error.
// If none of them matches, the error handling is delegating to the caller.
func HandleRequestFlowError(ctx context.Context, traceID string, interactor Interactor, err error) *jsonrpc.ErrorDetails {
	if errors.Is(err, ErrUserCancelledTheRequest) {
		// 1. Using a different error message to better fit the front-end needs.
		// 2. Contrary to the response returned to the third-party application,
		//    we notify an ApplicationErrorType to the wallet front-end, so it knows
		//    it has to consider this error as a terminal one.
		interactor.NotifyError(ctx, traceID, ApplicationErrorType, ErrRequestCancelled)
		return UserCancellationError(err)
	}
	if errors.Is(err, ErrUserCloseTheConnection) {
		// 1. Using a different error message to better fit the front-end needs.
		// 2. Contrary to the response returned to the third-party application,
		//    we notify an ApplicationErrorType to the wallet front-end, so it knows
		//    it has to consider this error as a terminal one.
		interactor.NotifyError(ctx, traceID, ApplicationErrorType, ErrConnectionClosed)
		return ConnectionClosedError(err)
	}
	if errors.Is(err, ErrRequestInterrupted) {
		interactor.NotifyError(ctx, traceID, ServerErrorType, err)
		return RequestInterruptedError(err)
	}
	return nil
}
