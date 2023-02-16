// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package admin

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/rpc/json"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

// Client implement a socket client allowing to run simple RPC commands.
type Client struct {
	log  *logging.Logger
	cfg  Config
	http *http.Client
}

// NewClient returns a new instance of the RPC socket client.
func NewClient(
	log *logging.Logger,
	config Config,
) *Client {
	// setup logger
	log = log.Named(clientNamedLogger)
	log.SetLevel(config.Level.Get())

	return &Client{
		log: log,
		cfg: config,
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", config.Server.SocketPath)
				},
			},
		},
	}
}

func (s *Client) call(ctx context.Context, method string, args interface{}, reply interface{}) error {
	req, err := json.EncodeClientRequest(method, args)
	if err != nil {
		return fmt.Errorf("failed to encode client JSON request: %w", err)
	}

	u := url.URL{
		Scheme: "http",
		Host:   "unix",
		Path:   s.cfg.Server.HTTPPath,
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(req))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to post data %q: %w", string(req), err)
	}
	defer resp.Body.Close()

	if err := json.DecodeClientResponse(resp.Body, reply); err != nil {
		return fmt.Errorf("failed to decode client JSON response: %w", err)
	}

	return nil
}

func (s *Client) NodeWalletReload(ctx context.Context, chain string) (*NodeWalletReloadReply, error) {
	var reply NodeWalletReloadReply

	if err := s.call(ctx, "NodeWallet.Reload", NodeWalletArgs{Chain: chain}, &reply); err != nil {
		return nil, fmt.Errorf("failed to call NodeWallet.Reload method: %w", err)
	}

	return &reply, nil
}

func (s *Client) NodeWalletShow(ctx context.Context, chain string) (*Wallet, error) {
	var reply Wallet

	if err := s.call(ctx, "NodeWallet.Show", NodeWalletArgs{Chain: chain}, &reply); err != nil {
		return nil, fmt.Errorf("failed to call NodeWallet.Show method: %w", err)
	}

	return &reply, nil
}

func (s *Client) UpgradeStatus(ctx context.Context) (*types.UpgradeStatus, error) {
	var reply types.UpgradeStatus

	if err := s.call(ctx, "protocolupgrade.UpgradeStatus", nil, &reply); err != nil {
		return nil, fmt.Errorf("failed to call protocolupgrade.UpgradeStatus method: %w", err)
	}

	return &reply, nil
}
