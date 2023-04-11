package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/preferences"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

const (
	WalletConnectionSuccessfullyEstablished = "The connection to the wallet has been successfully established."
	PassphraseRequestReasonUnlockWallet     = "The application wants to unlock the wallet."
)

type ClientConnectWallet struct {
	walletStore WalletStore
	interactor  Interactor
}

// Handle initiates the wallet connection between the API and a third-party
// application.
//
// It triggers a selection of the wallet the client wants to use for this
// connection. The wallet is then loaded in memory. All changes done to that wallet
// will start in-memory, and then, be saved in the wallet file. Any changes done
// to the wallet outside the JSON-RPC session (via the command-line for example)
// will be overridden. For the effects to be taken into account, the wallet has
// to be disconnected first, and then re-connected.
//
// All sessions have to be initialized by using this handler. Otherwise, a call
// to any other handlers will be rejected.
func (h *ClientConnectWallet) Handle(ctx context.Context, hostname string) (wallet.Wallet, *jsonrpc.ErrorDetails) {
	traceID := jsonrpc.TraceIDFromContext(ctx)

	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID, WalletConnectionWorkflow, 4); err != nil {
		return nil, RequestNotPermittedError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	availableWallets, err := h.walletStore.ListWallets(ctx)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not list the available wallets: %w", err))
		return nil, InternalError(ErrCouldNotConnectToWallet)
	}
	if len(availableWallets) == 0 {
		h.interactor.NotifyError(ctx, traceID, ApplicationErrorType, ErrNoWalletToConnectTo)
		return nil, ApplicationCancellationError(ErrApplicationCancelledTheRequest)
	}

	var approval preferences.ConnectionApproval
	for {
		rawApproval, err := h.interactor.RequestWalletConnectionReview(ctx, traceID, 1, hostname)
		if err != nil {
			if errDetails := HandleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
				return nil, errDetails
			}
			h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("reviewing the wallet connection failed: %w", err))
			return nil, InternalError(ErrCouldNotConnectToWallet)
		}

		a, err := preferences.ParseConnectionApproval(rawApproval)
		if err != nil {
			h.interactor.NotifyError(ctx, traceID, UserErrorType, err)
			continue
		}
		approval = a
		break
	}

	if isConnectionRejected(approval) {
		return nil, UserRejectionError(ErrUserRejectedWalletConnection)
	}

	var walletName string
	if len(availableWallets) > 1 {
		for {
			if ctx.Err() != nil {
				h.interactor.NotifyError(ctx, traceID, ApplicationErrorType, ErrRequestInterrupted)
				return nil, RequestInterruptedError(ErrRequestInterrupted)
			}

			selectedWallet, err := h.interactor.RequestWalletSelection(ctx, traceID, 2, hostname, availableWallets)
			if err != nil {
				if errDetails := HandleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
					return nil, errDetails
				}
				h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("requesting the wallet selection failed: %w", err))
				return nil, InternalError(ErrCouldNotConnectToWallet)
			}

			if exist, err := h.walletStore.WalletExists(ctx, selectedWallet); err != nil {
				h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not verify the wallet existence: %w", err))
				return nil, InternalError(ErrCouldNotConnectToWallet)
			} else if !exist {
				h.interactor.NotifyError(ctx, traceID, UserErrorType, ErrWalletDoesNotExist)
				continue
			}

			walletName = selectedWallet
			break
		}
	} else {
		// There is single wallet available, it doesn't make sense to ask which
		// wallet to use.
		walletName = availableWallets[0]
	}

	alreadyUnlocked, err := h.walletStore.IsWalletAlreadyUnlocked(ctx, walletName)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not verify whether the wallet is already unlock or not: %w", err))
		return nil, InternalError(ErrCouldNotConnectToWallet)
	}

	if !alreadyUnlocked {
		for {
			if ctx.Err() != nil {
				h.interactor.NotifyError(ctx, traceID, ApplicationErrorType, ErrRequestInterrupted)
				return nil, RequestInterruptedError(ErrRequestInterrupted)
			}

			walletPassphrase, err := h.interactor.RequestPassphrase(ctx, traceID, 3, walletName, PassphraseRequestReasonUnlockWallet)
			if err != nil {
				if errDetails := HandleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
					return nil, errDetails
				}
				h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("requesting the wallet passphrase failed: %w", err))
				return nil, InternalError(ErrCouldNotConnectToWallet)
			}

			if err := h.walletStore.UnlockWallet(ctx, walletName, walletPassphrase); err != nil {
				if errors.Is(err, wallet.ErrWrongPassphrase) {
					h.interactor.NotifyError(ctx, traceID, UserErrorType, wallet.ErrWrongPassphrase)
					continue
				}
				if errDetails := HandleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
					return nil, errDetails
				}
				h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not unlock the wallet: %w", err))
				return nil, InternalError(ErrCouldNotConnectToWallet)
			}
			break
		}
	}

	w, err := h.walletStore.GetWallet(ctx, walletName)
	if err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalErrorType, fmt.Errorf("could not retrieve the wallet: %w", err))
		return nil, InternalError(ErrCouldNotConnectToWallet)
	}

	h.interactor.NotifySuccessfulRequest(ctx, traceID, 4, WalletConnectionSuccessfullyEstablished)

	return w, nil
}

func isConnectionRejected(approval preferences.ConnectionApproval) bool {
	return approval != preferences.ApprovedOnlyThisTime
}

func NewConnectWallet(walletStore WalletStore, interactor Interactor) *ClientConnectWallet {
	return &ClientConnectWallet{
		walletStore: walletStore,
		interactor:  interactor,
	}
}
