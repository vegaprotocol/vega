package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type RequestPermissionsParams struct {
	Token                string                    `json:"token"`
	RequestedPermissions wallet.PermissionsSummary `json:"requestedPermissions"`
}

type RequestPermissionsResult struct {
	Permissions wallet.PermissionsSummary `json:"permissions"`
}

type RequestPermissions struct {
	walletStore WalletStore
	pipeline    Pipeline
	sessions    *Sessions
}

// Handle allows a third-party application to request permissions to access
// certain capabilities of the wallet.
//
// To update the permissions, the third-party application has to specify all
// the permissions it required, even those that are already active. This way the
// client get a full understanding of all the requested access, and is much more
// capable to evaluate abusive requests and applications. Any permission that is
// omitted is considered to be revoked.
//
// The client will be asked to review the permissions the third-party application
// is requesting. It has the possibility to approve or reject the request.
//
// Everytime the permissions are updated, the connected wallet resources are
// updated.
//
// Using this handler does not require permissions.
func (h *RequestPermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	traceID := TraceIDFromContext(ctx)

	params, err := validateRequestPermissionsParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	connectedWallet, err := h.sessions.GetConnectedWallet(params.Token)
	if err != nil {
		return nil, invalidParams(err)
	}

	approved, err := h.pipeline.RequestPermissionsReview(ctx, traceID, connectedWallet.Hostname, connectedWallet.Wallet.Name(), params.RequestedPermissions)
	if err != nil {
		if errDetails := handleRequestFlowError(ctx, traceID, h.pipeline, err); errDetails != nil {
			return nil, errDetails
		}
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the permissions review failed: %w", err))
		return nil, internalError(ErrCouldNotRequestPermissions)
	}
	if !approved {
		return nil, clientRejectionError()
	}

	perms, err := h.parsePermissions(params)
	if err != nil {
		return nil, invalidParams(err)
	}

	var passphrase string
	var walletFromStore wallet.Wallet
	for {
		if ctx.Err() != nil {
			return nil, requestInterruptedError(ErrRequestInterrupted)
		}

		enteredPassphrase, err := h.pipeline.RequestPassphrase(ctx, traceID, connectedWallet.Wallet.Name())
		if err != nil {
			if errDetails := handleRequestFlowError(ctx, traceID, h.pipeline, err); errDetails != nil {
				return nil, errDetails
			}
			h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("requesting the passphrase failed: %w", err))
			return nil, internalError(ErrCouldNotRequestPermissions)
		}

		w, err := h.walletStore.GetWallet(ctx, connectedWallet.Wallet.Name(), enteredPassphrase)
		if err != nil {
			if errors.Is(err, wallet.ErrWrongPassphrase) {
				h.pipeline.NotifyError(ctx, traceID, ClientError, wallet.ErrWrongPassphrase)
				continue
			}
			h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not retrieve the wallet: %w", err))
			return nil, internalError(ErrCouldNotRequestPermissions)
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
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not update the permissions: %w", err))
		return nil, internalError(ErrCouldNotRequestPermissions)
	}

	// Then, we update the in-memory wallet with the updated wallet, before
	// saving it, to ensure there is no problem with the resources reloading.
	if err := connectedWallet.ReloadWithWallet(walletFromStore); err != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not reload wallet's resources: %w", err))
		return nil, internalError(ErrCouldNotRequestPermissions)
	}

	// And, to finish, we save the wallet loaded from the wallet store.
	if err := h.walletStore.SaveWallet(ctx, walletFromStore, passphrase); err != nil {
		// We ignore the error as we know the previous state worked so far.
		// There is no sane reason it fails out of the blue.
		_ = connectedWallet.ReloadWithWallet(previousWallet)

		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("could not save the wallet: %w", err))
		return nil, internalError(ErrCouldNotRequestPermissions)
	}

	h.pipeline.NotifySuccessfulRequest(ctx, traceID)

	return RequestPermissionsResult{
		Permissions: perms.Summary(),
	}, nil
}

func (h *RequestPermissions) parsePermissions(params RequestPermissionsParams) (wallet.Permissions, error) {
	perms := wallet.Permissions{}

	if err := h.extractPublicKeysPermission(&perms, params); err != nil {
		return wallet.Permissions{}, err
	}

	return perms, nil
}

func (h *RequestPermissions) extractPublicKeysPermission(detailedPerms *wallet.Permissions, params RequestPermissionsParams) error {
	access, ok := params.RequestedPermissions[wallet.PublicKeysPermissionLabel]
	if !ok {
		// If the public keys permissions is omitted, we revoke it.
		detailedPerms.PublicKeys = wallet.NoPublicKeysPermission()
		return nil
	}

	// The correctness of the access mode should be valid at this point, but
	// we never know.
	mode, err := wallet.ToAccessMode(access)
	if err != nil {
		return err
	}

	// An access explicitly set to none is understood as a revocation.
	if mode == wallet.NoAccess {
		detailedPerms.PublicKeys = wallet.NoPublicKeysPermission()
		return nil
	}

	// TODO(valentin) Add future restricted key selection here

	detailedPerms.PublicKeys = wallet.PublicKeysPermission{
		Access: mode,
		// We don't yet support restricting the list of keys a third-party application
		// will have access to. Nil means access to all keys.
		RestrictedKeys: nil,
	}
	return nil
}

func validateRequestPermissionsParams(rawParams jsonrpc.Params) (RequestPermissionsParams, error) {
	if rawParams == nil {
		return RequestPermissionsParams{}, ErrParamsRequired
	}

	params := RequestPermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return RequestPermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return RequestPermissionsParams{}, ErrConnectionTokenIsRequired
	}

	if len(params.RequestedPermissions) == 0 {
		return RequestPermissionsParams{}, ErrRequestedPermissionsAreRequired
	}

	for permission, mode := range params.RequestedPermissions {
		if !isSupportedPermissions(permission) {
			return RequestPermissionsParams{}, fmt.Errorf("permission %q is not supported", permission)
		}
		if !isSupportedAccessMode(mode) {
			return RequestPermissionsParams{}, fmt.Errorf("access mode %q is not supported", mode)
		}
	}

	return params, nil
}

func NewRequestPermissions(
	walletStore WalletStore,
	pipeline Pipeline,
	sessions *Sessions,
) *RequestPermissions {
	return &RequestPermissions{
		walletStore: walletStore,
		pipeline:    pipeline,
		sessions:    sessions,
	}
}

var supportedPermissions = []string{
	wallet.PublicKeysPermissionLabel,
}

func isSupportedPermissions(perm string) bool {
	for _, supportedPermissions := range supportedPermissions {
		if perm == supportedPermissions {
			return true
		}
	}
	return false
}

var supportedAccessMode = []string{
	wallet.AccessModeToString(wallet.NoAccess),
	wallet.AccessModeToString(wallet.ReadAccess),
	wallet.AccessModeToString(wallet.WriteAccess),
}

func isSupportedAccessMode(mode string) bool {
	for _, supportedMode := range supportedAccessMode {
		if mode == supportedMode {
			return true
		}
	}
	return false
}
