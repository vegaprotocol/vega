package v2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"go.uber.org/zap"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/wallet/service/v2 ClientAPI

type ClientAPI interface {
	ConnectWallet(ctx context.Context, hostname string) (wallet.Wallet, *jsonrpc.ErrorDetails)
	GetChainID(ctx context.Context) (jsonrpc.Result, *jsonrpc.ErrorDetails)
	ListKeys(ctx context.Context, connectedWallet api.ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails)
	CheckTransaction(ctx context.Context, params jsonrpc.Params, connectedWallet api.ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails)
	SignTransaction(ctx context.Context, rawParams jsonrpc.Params, connectedWallet api.ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails)
	SendTransaction(ctx context.Context, rawParams jsonrpc.Params, connectedWallet api.ConnectedWallet) (jsonrpc.Result, *jsonrpc.ErrorDetails)
}

type Command func(ctx context.Context, lw *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails)

type API struct {
	log *zap.Logger

	commands map[string]Command
}

// NewAPI builds the wallet JSON-RPC API with specific methods that are
// intended to be publicly exposed to third-party applications in a
// non-trustable environment.
// Because of the nature of the environment from where these methods are called,
// (the "wild, wild web"), no administration methods are exposed. We don't want
// malicious third-party applications to leverage administration capabilities
// that could expose the user and/or compromise his wallets.
func NewAPI(log *zap.Logger, clientAPI ClientAPI, connectionsManager *connections.Manager) *API {
	commands := map[string]Command{}

	commands["client.connect_wallet"] = func(ctx context.Context, lw *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
		hostname, err := resolveHostname(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeHostnameResolutionFailure, err)
		}

		selectedWallet, errDetails := clientAPI.ConnectWallet(ctx, hostname)
		if errDetails != nil {
			return nil, errDetails
		}

		token, err := connectionsManager.StartSession(hostname, selectedWallet)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, err)
		}

		lw.SetAuthorization(AsVWT(token))

		return nil, nil
	}

	commands["client.disconnect_wallet"] = func(ctx context.Context, _ *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
		vwt, err := ExtractVWT(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, err)
		}

		connectionsManager.EndSessionConnectionWithToken(vwt.Token())

		return nil, nil
	}

	commands["client.get_chain_id"] = func(ctx context.Context, _ *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
		return clientAPI.GetChainID(ctx)
	}

	commands["client.list_keys"] = func(ctx context.Context, _ *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
		hostname, err := resolveHostname(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeHostnameResolutionFailure, err)
		}

		vwt, err := ExtractVWT(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, err)
		}

		connectedWallet, errDetails := connectionsManager.ConnectedWallet(ctx, hostname, vwt.Token())
		if errDetails != nil {
			return nil, errDetails
		}

		return clientAPI.ListKeys(ctx, connectedWallet)
	}

	commands["client.sign_transaction"] = func(ctx context.Context, _ *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
		hostname, err := resolveHostname(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeHostnameResolutionFailure, err)
		}

		vwt, err := ExtractVWT(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, err)
		}

		connectedWallet, errDetails := connectionsManager.ConnectedWallet(ctx, hostname, vwt.Token())
		if errDetails != nil {
			return nil, errDetails
		}

		return clientAPI.SignTransaction(ctx, rpcRequest.Params, connectedWallet)
	}

	commands["client.send_transaction"] = func(ctx context.Context, _ *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
		hostname, err := resolveHostname(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeHostnameResolutionFailure, err)
		}

		vwt, err := ExtractVWT(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, err)
		}

		connectedWallet, errDetails := connectionsManager.ConnectedWallet(ctx, hostname, vwt.Token())
		if errDetails != nil {
			return nil, errDetails
		}

		return clientAPI.SendTransaction(ctx, rpcRequest.Params, connectedWallet)
	}

	commands["client.check_transaction"] = func(ctx context.Context, _ *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
		hostname, err := resolveHostname(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeHostnameResolutionFailure, err)
		}

		vwt, err := ExtractVWT(httpRequest)
		if err != nil {
			return nil, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, err)
		}

		connectedWallet, errDetails := connectionsManager.ConnectedWallet(ctx, hostname, vwt.Token())
		if errDetails != nil {
			return nil, errDetails
		}

		return clientAPI.CheckTransaction(ctx, rpcRequest.Params, connectedWallet)
	}

	return &API{
		log:      log,
		commands: commands,
	}
}

// resolveHostname attempts to resolve the source of the request by parsing the
// Origin (and if absent, the Referer) header.
// If it fails to resolve the hostname, it returns an error.
func resolveHostname(r *http.Request) (string, error) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			return origin, nil //nolint:nilerr
		}
		if parsedOrigin.Host != "" {
			return parsedOrigin.Host, nil
		}
		return normalizeHostname(origin)
	}

	// In some scenario, the Origin can be set to null by the browser for privacy
	// reasons. Since we are not trying to fingerprint or spoof anyone, we
	// attempt to parse the Referer.
	referer := r.Header.Get("Referer")
	if referer != "" {
		parsedReferer, err := url.Parse(referer)
		if err != nil {
			return "", fmt.Errorf("could not parse the Referer header: %w", err)
		}
		return normalizeHostname(parsedReferer.Host)
	}

	// If none of the Origin and Referer headers are present, we just report that
	// the missing Origin header as we should, ideally, only rely on this header.
	// The Referer is just a "desperate" attempt to get information about the
	// origin of the request and minimize the friction with future privacy
	// settings.
	return "", ErrOriginHeaderIsRequired
}

func normalizeHostname(host string) (string, error) {
	host = trimBlankCharacters(host)

	if host == "" {
		return "", ErrOriginHeaderIsRequired
	}

	return host, nil
}

func trimBlankCharacters(host string) string {
	return strings.Trim(host, " \r\n\t")
}
