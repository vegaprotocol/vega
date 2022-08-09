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

		_, err = h.walletStore.GetWallet(ctx, connectedWallet.Wallet.Name(), enteredPassphrase)
		if err != nil {
			if errors.Is(err, wallet.ErrWrongPassphrase) {
				h.pipeline.NotifyError(ctx, traceID, ClientError, wallet.ErrWrongPassphrase)
				continue
			}
			h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("couldn't retrieve the wallet: %w", err))
			return nil, internalError(ErrCouldNotRequestPermissions)
		}
		passphrase = enteredPassphrase
		break
	}

	previousPerms := connectedWallet.Permissions()
	var updateErr error
	defer func() {
		// If any of the actions below fails, we try to revert the in-memory
		// wallet permissions to the previous state.
		// It may fail, but at least we tried.
		if updateErr != nil {
			_ = connectedWallet.Wallet.UpdatePermissions(connectedWallet.Hostname, previousPerms)
		}
	}()
	if updateErr = connectedWallet.UpdatePermissions(perms); updateErr != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("couldn't update the permissions: %w", updateErr))
		return nil, internalError(ErrCouldNotRequestPermissions)
	}

	if updateErr = h.walletStore.SaveWallet(ctx, connectedWallet.Wallet, passphrase); updateErr != nil {
		h.pipeline.NotifyError(ctx, traceID, InternalError, fmt.Errorf("couldn't save wallet: %w", updateErr))
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
