package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type ListKeysParams struct {
	Token string `json:"token"`
}

type ListKeysResult struct {
	Keys []string `json:"keys"`
}

type ListKeys struct {
	sessions *Sessions
}

// Handle returns the public keys the third-party application has access to.
//
// This requires a "read" access on "public_keys".
func (h *ListKeys) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateListKeysParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	connectedWallet, err := h.sessions.GetConnectedWallet(params.Token)
	if err != nil {
		return nil, invalidParams(err)
	}

	if !connectedWallet.Permissions().CanListKeys() {
		return nil, requestNotPermittedError(ErrReadAccessOnPublicKeysRequired)
	}

	keys := make([]string, 0, len(connectedWallet.RestrictedKeys))
	for pubKey := range connectedWallet.RestrictedKeys {
		keys = append(keys, pubKey)
	}

	return ListKeysResult{
		Keys: keys,
	}, nil
}

func validateListKeysParams(rawParams jsonrpc.Params) (ListKeysParams, error) {
	if rawParams == nil {
		return ListKeysParams{}, ErrParamsRequired
	}

	params := ListKeysParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ListKeysParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return ListKeysParams{}, ErrConnectionTokenIsRequired
	}

	return params, nil
}

func NewListKeys(sessions *Sessions) *ListKeys {
	return &ListKeys{
		sessions: sessions,
	}
}
