package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/preferences"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

const WalletConnectionSuccessfullyEstablished = "The connection to the wallet has been successfully established."

type ClientConnectWallet struct {
	walletStore WalletStore
	interactor  Interactor
	sessions    *session.Sessions
}

type ClientConnectWalletResult struct {
	Token string `json:"token"`
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
func (h *ClientConnectWallet) Handle(ctx context.Context, _ jsonrpc.Params, metadata jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	if err := h.interactor.NotifyInteractionSessionBegan(ctx, metadata.TraceID); err != nil {
		return nil, internalError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, metadata.TraceID)

	availableWallets, err := h.walletStore.ListWallets(ctx)
	if err != nil {
		h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("could not list the available wallets: %w", err))
		return nil, internalError(ErrCouldNotConnectToWallet)
	}
	if len(availableWallets) == 0 {
		h.interactor.NotifyError(ctx, metadata.TraceID, ApplicationError, ErrNoWalletToConnectTo)
		return nil, applicationCancellationError(ErrApplicationCanceledTheRequest)
	}

	var approval preferences.ConnectionApproval
	for {
		rawApproval, err := h.interactor.RequestWalletConnectionReview(ctx, metadata.TraceID, metadata.Hostname)
		if err != nil {
			if errDetails := handleRequestFlowError(ctx, metadata.TraceID, h.interactor, err); errDetails != nil {
				return nil, errDetails
			}
			h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("reviewing the wallet connection failed: %w", err))
			return nil, internalError(ErrCouldNotConnectToWallet)
		}

		a, err := preferences.ParseConnectionApproval(rawApproval)
		if err != nil {
			h.interactor.NotifyError(ctx, metadata.TraceID, UserError, err)
			continue
		}
		approval = a
		break
	}

	if isConnectionRejected(approval) {
		return nil, userRejectionError()
	}

	// Wallet selection process.
	var loadedWallet wallet.Wallet
	for {
		if ctx.Err() != nil {
			return nil, requestInterruptedError(ErrRequestInterrupted)
		}

		selectedWallet, err := h.interactor.RequestWalletSelection(ctx, metadata.TraceID, metadata.Hostname, availableWallets)
		if err != nil {
			if errDetails := handleRequestFlowError(ctx, metadata.TraceID, h.interactor, err); errDetails != nil {
				return nil, errDetails
			}
			h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("requesting the wallet selection failed: %w", err))
			return nil, internalError(ErrCouldNotConnectToWallet)
		}

		if exist, err := h.walletStore.WalletExists(ctx, selectedWallet.Wallet); err != nil {
			h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("could not verify the wallet existence: %w", err))
			return nil, internalError(ErrCouldNotConnectToWallet)
		} else if !exist {
			h.interactor.NotifyError(ctx, metadata.TraceID, UserError, ErrWalletDoesNotExist)
			continue
		}

		w, err := h.walletStore.GetWallet(ctx, selectedWallet.Wallet, selectedWallet.Passphrase)
		if err != nil {
			if errors.Is(err, wallet.ErrWrongPassphrase) {
				h.interactor.NotifyError(ctx, metadata.TraceID, UserError, wallet.ErrWrongPassphrase)
				continue
			}
			h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("could not retrieve the wallet: %w", err))
			return nil, internalError(ErrCouldNotConnectToWallet)
		}
		loadedWallet = w
		break
	}

	token, err := h.sessions.ConnectWallet(metadata.Hostname, loadedWallet)
	if err != nil {
		h.interactor.NotifyError(ctx, metadata.TraceID, InternalError, fmt.Errorf("could not connect to a wallet: %w", err))
		return nil, internalError(ErrCouldNotConnectToWallet)
	}

	h.interactor.NotifySuccessfulRequest(ctx, metadata.TraceID, WalletConnectionSuccessfullyEstablished)

	return ClientConnectWalletResult{
		Token: token,
	}, nil
}

func isConnectionRejected(approval preferences.ConnectionApproval) bool {
	return approval != preferences.ApprovedOnlyThisTime
}

func NewConnectWallet(walletStore WalletStore, interactor Interactor, sessions *session.Sessions) *ClientConnectWallet {
	return &ClientConnectWallet{
		walletStore: walletStore,
		interactor:  interactor,
		sessions:    sessions,
	}
}
