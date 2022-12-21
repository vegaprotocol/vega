package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

const PermissionsSuccessfullyUpdated = "The permissions have been successfully updated."

type ClientListKeysParams struct {
	Token string `json:"token"`
}

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
	sessions    *session.Sessions
}

// Handle returns the public keys the third-party application has access to.
//
// This requires a "read" access on "public_keys".
func (h *ClientListKeys) Handle(ctx context.Context, rawParams jsonrpc.Params, metadata jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateSessionListKeysParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	connectedWallet, err := h.sessions.GetConnectedWallet(params.Token, time.Now())
	if err != nil {
		return nil, invalidParams(err)
	}

	if perms := connectedWallet.Permissions(); !perms.CanListKeys() {
		// we need to now ask for read permissions
		perms.PublicKeys.Access = wallet.ReadAccess
		if err := h.requestPermissions(ctx, metadata.TraceID, connectedWallet, perms); err != nil {
			return nil, err
		}
	}

	keys := make([]ClientNamedPublicKey, 0, len(connectedWallet.RestrictedKeys))

	for _, keyPair := range connectedWallet.RestrictedKeys {
		keys = append(keys, ClientNamedPublicKey{
			Name:      keyPair.Name(),
			PublicKey: keyPair.PublicKey(),
		})
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i].PublicKey < keys[j].PublicKey })

	return ClientListKeysResult{
		Keys: keys,
	}, nil
}

func (h *ClientListKeys) requestPermissions(ctx context.Context, traceID string, connectedWallet *session.ConnectedWallet, perms wallet.Permissions) *jsonrpc.ErrorDetails {
	if err := h.interactor.NotifyInteractionSessionBegan(ctx, traceID); err != nil {
		return internalError(err)
	}
	defer h.interactor.NotifyInteractionSessionEnded(ctx, traceID)

	approved, err := h.interactor.RequestPermissionsReview(ctx, traceID, connectedWallet.Hostname, connectedWallet.Wallet.Name(), perms.Summary())
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
			return errDetails
		}
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the permissions review failed: %w", err))
		return internalError(ErrCouldNotRequestPermissions)
	}
	if !approved {
		return userRejectionError()
	}

	var passphrase string
	var walletFromStore wallet.Wallet
	for {
		if ctx.Err() != nil {
			return requestInterruptedError(ErrRequestInterrupted)
		}

		enteredPassphrase, err := h.interactor.RequestPassphrase(ctx, traceID, connectedWallet.Wallet.Name())
		if err != nil {
			if errDetails := handleRequestFlowError(ctx, traceID, h.interactor, err); errDetails != nil {
				return errDetails
			}
			h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the passphrase failed: %w", err))
			return internalError(ErrCouldNotRequestPermissions)
		}

		w, err := h.walletStore.GetWallet(ctx, connectedWallet.Wallet.Name(), enteredPassphrase)
		if err != nil {
			if errors.Is(err, wallet.ErrWrongPassphrase) {
				h.interactor.NotifyError(ctx, traceID, UserError, wallet.ErrWrongPassphrase)
				continue
			}
			h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not retrieve the wallet: %w", err))
			return internalError(ErrCouldNotRequestPermissions)
		}
		passphrase = enteredPassphrase
		walletFromStore = w
		break
	}

	// We keep a reference to the in-memory wallet, it case we need to roll back.
	previousWallet := connectedWallet.Wallet

	// We update the wallet we just loaded from the wallet store to ensure
	// we don't overwrite changes that could have been done outside the API.
	if err := walletFromStore.UpdatePermissions(connectedWallet.Hostname, perms); err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not update the permissions: %w", err))
		return internalError(ErrCouldNotRequestPermissions)
	}

	// Then, we update the in-memory wallet with the updated wallet, before
	// saving it, to ensure there is no problem with the resources reloading.
	if err := connectedWallet.ReloadWithWallet(walletFromStore); err != nil {
		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not reload wallet's resources: %w", err))
		return internalError(ErrCouldNotRequestPermissions)
	}

	// And, to finish, we save the wallet loaded from the wallet store.
	if err := h.walletStore.SaveWallet(ctx, walletFromStore, passphrase); err != nil {
		// We ignore the error as we know the previous state worked so far.
		// There is no sane reason it fails out of the blue.
		_ = connectedWallet.ReloadWithWallet(previousWallet)

		h.interactor.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not save the wallet: %w", err))
		return internalError(ErrCouldNotRequestPermissions)
	}

	h.interactor.NotifySuccessfulRequest(ctx, traceID, PermissionsSuccessfullyUpdated)
	return nil
}

func validateSessionListKeysParams(rawParams jsonrpc.Params) (ClientListKeysParams, error) {
	if rawParams == nil {
		return ClientListKeysParams{}, ErrParamsRequired
	}

	params := ClientListKeysParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ClientListKeysParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return ClientListKeysParams{}, ErrConnectionTokenIsRequired
	}

	return params, nil
}

func NewListKeys(walletStore WalletStore, interactor Interactor, sessions *session.Sessions) *ClientListKeys {
	return &ClientListKeys{
		walletStore: walletStore,
		interactor:  interactor,
		sessions:    sessions,
	}
}
