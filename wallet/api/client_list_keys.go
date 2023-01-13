package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

const PermissionsSuccessfullyUpdated = "The permissions have been successfully updated."

type ClientListKeysResult struct {
	Keys []ClientNamedPublicKey `json:"keys"`
}

type ClientNamedPublicKey struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

type ClientListKeys struct {
	walletStore WalletStore
	interactor  Interactor
}

// Handle returns the public keys the third-party application has access to.
//
// This requires a "read" access on "public_keys".
func (h *ClientListKeys) Handle(ctx context.Context, connectedWallet ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := jsonrpc.TraceIDFromContext(ctx)

	if !connectedWallet.CanListKeys() {
		if err := h.updatePublicKeysPermissions(ctx, traceID, &connectedWallet); err != nil {
			return nil, err
		}
	}

	restrictedKeys := connectedWallet.RestrictedKeys()

	keys := make([]ClientNamedPublicKey, 0, len(restrictedKeys))
	for _, restrictedKey := range restrictedKeys {
		keys = append(keys, ClientNamedPublicKey{
			Name:      restrictedKey.Name(),
			PublicKey: restrictedKey.PublicKey(),
		})
	}

	return ClientListKeysResult{
		Keys: keys,
	}, nil
}

func (h *ClientListKeys) updatePublicKeysPermissions(ctx context.Context, traceID string, connectedWallet *ConnectedWallet) *jsonrpc.ErrorDetails {
	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID); err != nil {
		return requestNotPermittedError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	freshWallet, err := h.walletStore.GetWallet(ctx, connectedWallet.Name())
	if err != nil {
		if errors.Is(err, ErrWalletIsLocked) {
			h.interactor.NotifyError(ctx, traceID, ApplicationError, err)
		} else {
			h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not retrieve the wallet for the permissions update: %w", err))
		}
		return internalError(ErrCouldNotListKeys)
	}

	perms := freshWallet.Permissions(connectedWallet.Hostname())

	// At this point, we need a "read" access on public keys.
	perms.PublicKeys.Access = wallet.ReadAccess
	approved, err := h.interactor.RequestPermissionsReview(ctx, traceID, connectedWallet.Hostname(), connectedWallet.Name(), perms.Summary())
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
			return errDetails
		}
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the permissions review failed: %w", err))
		return internalError(ErrCouldNotListKeys)
	}
	if !approved {
		return userRejectionError()
	}

	if err := freshWallet.UpdatePermissions(connectedWallet.Hostname(), perms); err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not update the permissions on the wallet: %w", err))
		return internalError(ErrCouldNotListKeys)
	}

	if err := connectedWallet.RefreshFromWallet(freshWallet); err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not refresh the connection information after the permissions update: %w", err))
		return internalError(ErrCouldNotListKeys)
	}

	if err := h.walletStore.UpdateWallet(ctx, freshWallet); err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not save the permissions update on the wallet: %w", err))
		return internalError(ErrCouldNotListKeys)
	}

	h.interactor.NotifySuccessfulRequest(ctx, traceID, PermissionsSuccessfullyUpdated)

	return nil
}

func NewListKeys(walletStore WalletStore, interactor Interactor) *ClientListKeys {
	return &ClientListKeys{
		walletStore: walletStore,
		interactor:  interactor,
	}
}
