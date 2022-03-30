package admin

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"code.vegaprotocol.io/vega/logging"
	"github.com/gorilla/rpc/json"
)

// Client implement a socket client allowing to run simple RPC commands
type Client struct {
	log  *logging.Logger
	cfg  Config
	http *http.Client
}

// Client returns a new instance of the RPC socket client.
func NewClient(
	log *logging.Logger,
	config Config,
) *Client {
	// setup logger
	log = log.Named(namedLogger)
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

func (s *Client) call(method string, args any, reply any) error {
	req, err := json.EncodeClientRequest(method, args)
	if err != nil {
		return fmt.Errorf("failed to encode client JSON request: %w", err)
	}

	u := url.URL{
		Scheme: "http",
		Host:   "unix",
		Path:   s.cfg.Server.HttpPath,
	}

	resp, err := s.http.Post(u.String(), "application/json", bytes.NewReader(req))
	if err != nil {
		return fmt.Errorf("failed to post data %q: %w", string(req), err)
	}

	defer resp.Body.Close()

	if err := json.DecodeClientResponse(resp.Body, reply); err != nil {
		return fmt.Errorf("failed to decode client JSON response: %w", err)
	}

	return nil
}

func (s *Client) NodeWalletReload(chain string) (*NodeWalletReloadReply, error) {
	var reply NodeWalletReloadReply

	if err := s.call("NodeWallet.Reload", NodeWalletArgs{Chain: chain}, &reply); err != nil {
		return nil, fmt.Errorf("failed to call NodeWallet.Reload method: %w", err)
	}

	return &reply, nil
}

func (s *Client) NodeWalletShow(chain string) (*Wallet, error) {
	var reply Wallet

	if err := s.call("NodeWallet.Show", NodeWalletArgs{Chain: chain}, &reply); err != nil {
		return nil, fmt.Errorf("failed to call NodeWallet.Show method: %w", err)
	}

	return &reply, nil
}
