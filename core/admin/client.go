// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package admin

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

	"github.com/gorilla/rpc/json"
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
