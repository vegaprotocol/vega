package api

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type SessionListKeysParams struct {
	Token string `json:"token"`
}

type SessionListKeysResult struct {
	Keys []SessionNamedPublicKey `json:"keys"`
}

type SessionNamedPublicKey struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

type SessionListKeys struct {
	sessions *Sessions
}

// Handle returns the public keys the third-party application has access to.
//
// This requires a "read" access on "public_keys".
func (h *SessionListKeys) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
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

	keys := make([]SessionNamedPublicKey, 0, len(connectedWallet.RestrictedKeys))

	for _, keyPair := range connectedWallet.RestrictedKeys {
		keys = append(keys, SessionNamedPublicKey{
			Name:      keyPair.Name(),
			PublicKey: keyPair.PublicKey(),
		})
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i].PublicKey < keys[j].PublicKey })

	return SessionListKeysResult{
		Keys: keys,
	}, nil
}

func validateSessionListKeysParams(rawParams jsonrpc.Params) (SessionListKeysParams, error) {
	if rawParams == nil {
		return SessionListKeysParams{}, ErrParamsRequired
	}

	params := SessionListKeysParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return SessionListKeysParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return SessionListKeysParams{}, ErrConnectionTokenIsRequired
	}

	return params, nil
}

func NewListKeys(sessions *Sessions) *SessionListKeys {
	return &SessionListKeys{
		sessions: sessions,
	}
}
