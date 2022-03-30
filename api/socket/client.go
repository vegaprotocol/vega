package socket

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/logging"
	"github.com/gorilla/rpc/json"
)

// SocketClient implement a socket client allowing to run simple RPC commands
type SocketClient struct {
	log  *logging.Logger
	cfg  api.Config
	http *http.Client
}

// NewSocketClient returns a new instance of the RPC socket client.
func NewSocketClient(
	log *logging.Logger,
	config api.Config,
) *SocketClient {

	// l, err := net.Listen("unix", s.cfg.Socket.FilePath)

	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &SocketClient{
		log: log,
		cfg: config,
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", config.Socket.FilePath)
				},
			},
		},
	}
}

func (s *SocketClient) call(method string, args any, reply any) error {
	req, err := json.EncodeClientRequest(method, args)
	if err != nil {
		return fmt.Errorf("failed to encode client JSON request: %w", err)
	}

	u := url.URL{
		Scheme: "http",
		Host:   "unix",
		Path:   s.cfg.Socket.HttpPath,
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

func (s *SocketClient) NodeWalletReload(chain string) (*NodeWalletReloadReply, error) {
	var reply NodeWalletReloadReply

	if err := s.call("NodeWallet.Reload", NodeWalletArgs{Chain: chain}, &reply); err != nil {
		return nil, fmt.Errorf("failed to call NodeWallet.Reload method: %w", err)
	}

	return &reply, nil
}

func (s *SocketClient) NodeWalletShow(chain string) (*Wallet, error) {
	var reply Wallet

	if err := s.call("NodeWallet.Show", NodeWalletArgs{Chain: chain}, &reply); err != nil {
		return nil, fmt.Errorf("failed to call NodeWallet.Show method: %w", err)
	}

	return &reply, nil
}
