package api

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

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
	sessions *Sessions
}

// Handle returns the public keys the third-party application has access to.
//
// This requires a "read" access on "public_keys".
func (h *ClientListKeys) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateSessionListKeysParams(rawParams)
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

func NewListKeys(sessions *Sessions) *ClientListKeys {
	return &ClientListKeys{
		sessions: sessions,
	}
}
