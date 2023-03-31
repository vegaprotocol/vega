package admin

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/logging"

	"github.com/gorilla/rpc/json"
)

type Client struct {
	log  *logging.Logger
	cfg  Config
	http *http.Client
}

func NewClient(
	log *logging.Logger,
	config Config,
) *Client {
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

func (s *Client) FetchNetworkHistorySegment(ctx context.Context, historySegmentID string) (store.SegmentIndexEntry, error) {
	var reply store.SegmentIndexEntry
	err := s.call(ctx, "networkhistory.FetchHistorySegment", historySegmentID, &reply)
	return reply, err
}

func (s *Client) CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, filePath string) (CopyHistorySegmentToFileReply, error) {
	var reply CopyHistorySegmentToFileReply
	err := s.call(ctx, "networkhistory.CopyHistorySegmentToFile", CopyHistorySegmentToFileArg{
		HistorySegmentID: historySegmentID,
		OutFile:          filePath,
	}, &reply)
	return reply, err
}
