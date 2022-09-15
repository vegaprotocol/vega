package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type ConnectWallet struct {
	walletStore WalletStore
	pipeline    Pipeline
	sessions    *Sessions
}

type ConnectWalletParams struct {
	Hostname string `json:"hostname"`
}

type ConnectWalletResult struct {
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
func (h *ConnectWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := TraceIDFromContext(ctx)

	params, err := validateConnectWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	approved, err := h.pipeline.RequestWalletConnectionReview(ctx, traceID, params.Hostname)
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.pipeline, err); errDetails != nil {
			return nil, errDetails
		}
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("reviewing the wallet connection failed: %w", err))
		return nil, internalError(ErrCouldNotConnectToWallet)
	}
	if !approved {
		return nil, userRejectionError()
	}

	availableWallets, err := h.walletStore.ListWallets(ctx)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not list available wallets: %w", err))
		return nil, internalError(ErrCouldNotConnectToWallet)
	}

	// Wallet selection process.
	var loadedWallet wallet.Wallet
	for {
		if ctx.Err() != nil {
			return nil, requestInterruptedError(ErrRequestInterrupted)
		}

		selectedWallet, err := h.pipeline.RequestWalletSelection(ctx, traceID, params.Hostname, availableWallets)
		if err != nil {
			if errDetails := handleRequestFlowError(ctx, traceID, h.pipeline, err); errDetails != nil {
				return nil, errDetails
			}
			h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the wallet selection failed: %w", err))
			return nil, internalError(ErrCouldNotConnectToWallet)
		}

		if exist, err := h.walletStore.WalletExists(ctx, selectedWallet.Wallet); err != nil {
			h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not verify the wallet existence: %w", err))
			return nil, internalError(ErrCouldNotConnectToWallet)
		} else if !exist {
			h.pipeline.NotifyError(ctx, traceID, UserError, ErrWalletDoesNotExist)
			continue
		}

		w, err := h.walletStore.GetWallet(ctx, selectedWallet.Wallet, selectedWallet.Passphrase)
		if err != nil {
			if errors.Is(err, wallet.ErrWrongPassphrase) {
				h.pipeline.NotifyError(ctx, traceID, UserError, wallet.ErrWrongPassphrase)
				continue
			}
			h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not retrieve the wallet: %w", err))
			return nil, internalError(ErrCouldNotConnectToWallet)
		}
		loadedWallet = w
		break
	}

	token, err := h.sessions.ConnectWallet(params.Hostname, loadedWallet)
	if err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not connect to a wallet: %w", err))
		return nil, internalError(ErrCouldNotConnectToWallet)
	}

	h.pipeline.NotifySuccessfulRequest(ctx, traceID)

	return ConnectWalletResult{
		Token: token,
	}, nil
}

func validateConnectWalletParams(rawParams jsonrpc.Params) (ConnectWalletParams, error) {
	if rawParams == nil {
		return ConnectWalletParams{}, ErrParamsRequired
	}

	params := ConnectWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ConnectWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Hostname == "" {
		return ConnectWalletParams{}, ErrHostnameIsRequired
	}

	return params, nil
}

func NewConnectWallet(
	walletStore WalletStore,
	pipeline Pipeline,
	sessions *Sessions,
) *ConnectWallet {
	return &ConnectWallet{
		walletStore: walletStore,
		pipeline:    pipeline,
		sessions:    sessions,
	}
}
