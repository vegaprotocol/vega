package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type DisconnectWallet struct {
	sessions *Sessions
}

type DisconnectWalletParams struct {
	Token string `json:"hostname"`
}

// Handle disconnect a wallet for a third-party application.
//
// It doesn't fail. If there's no connected wallet for the given token, nothing
// happens.
//
// The wallet resources are unloaded.
func (h *DisconnectWallet) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDisconnectWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	h.sessions.DisconnectWallet(params.Token)

	return nil, nil
}

func validateDisconnectWalletParams(rawParams jsonrpc.Params) (DisconnectWalletParams, error) {
	if rawParams == nil {
		return DisconnectWalletParams{}, ErrParamsRequired
	}

	params := DisconnectWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return DisconnectWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return DisconnectWalletParams{}, ErrConnectionTokenIsRequired
	}

	return params, nil
}

func NewDisconnectWallet(sessions *Sessions) *DisconnectWallet {
	return &DisconnectWallet{
		sessions: sessions,
	}
}
