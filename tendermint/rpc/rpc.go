// Package rpc implements a WebSocket RPC client interface to Tendermint.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"vega/log"

	"github.com/gorilla/websocket"
)

const (
	address          = "127.0.0.1:46657"
	endpoint         = "/websocket"
	handshakeTimeout = 10 * time.Second
	writeTimeout     = 30 * time.Second
)

// Errors returned by the Client.
var (
	ErrCheckTxFailed = errors.New("rpc: CheckTx failed during the AddTransaction call")
	ErrClosed        = errors.New("rpc: client has already been closed")
)

type opts map[string]interface{}

type request struct {
	ID      uint64          `json:"id,string"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	Error   *rpcError       `json:"error"`
	ID      uint64          `json:"id,string"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Data    string `json:"data"`
	Message string `json:"message"`
}

func (err rpcError) Error() string {
	if err.Data != "" {
		return fmt.Sprintf("%d (%s): %s", err.Code, err.Message, err.Data)
	}
	return fmt.Sprintf("%d (%s)", err.Code, err.Message)
}

// Client can be used to interface with Tendermint. Initialise the Client by
// calling the Connect method, and then call the appropriate methods
// corresponding to Tendermint's JSON-RPC API.
type Client struct {
	Address          string        // Defaults to 127.0.0.1:46657
	HandshakeTimeout time.Duration // Defaults to 10 seconds
	WriteTimeout     time.Duration // Defaults to 30 seconds
	closed           chan struct{}
	conn             *websocket.Conn
	connClosed       bool
	err              error
	lastID           uint64
	mu               sync.RWMutex // Protects the closed, connClosed, err, lastID and results struct fields.
	pending          chan *request
	results          map[uint64]chan *response
}

// AddTransaction corresponds to the Tendermint BroadcastTxSync call. It adds
// the given transaction data to Tendermint's mempool and returns data from the
// corresponding ABCI CheckTx call.
//
// If the given transaction data fails CheckTx, then the method will return
// ErrCheckTxFailed, and the caller can inspect the returned CheckTxResult for
// the ABCI app-specific error code and log message.
func (c *Client) AddTransaction(ctx context.Context, txdata []byte) (*CheckTxResult, error) {
	resp := &CheckTxResult{}
	if err := c.call(ctx, "broadcast_tx_sync", opts{"tx": txdata}, resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return resp, ErrCheckTxFailed
	}
	return resp, nil
}

// AsyncTransaction corresponds to the Tendermint BroadcastTxAsync call. It adds
// the given transaction data to Tendermint's mempool and returns immediately.
func (c *Client) AsyncTransaction(ctx context.Context, txdata []byte) error {
	return c.call(ctx, "broadcast_tx_async", opts{"tx": txdata}, nil)
}

// Close terminates the underlying WebSocket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeWithError(nil)
}

// Connect initialises the Client and establishes a WebSocket connection to the
// Tendermint endpoint. It must only be called once per Client.
func (c *Client) Connect() error {
	if c.Address == "" {
		c.Address = address
	}
	if c.HandshakeTimeout == 0 {
		c.HandshakeTimeout = handshakeTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = writeTimeout
	}
	dialer := &websocket.Dialer{
		HandshakeTimeout: c.HandshakeTimeout,
	}
	w, _, err := dialer.Dial("ws://"+c.Address+endpoint, nil)
	if err != nil {
		return err
	}
	pings := make(chan string, 100)
	w.SetPingHandler(func(m string) error {
		select {
		case pings <- m:
		default:
		}
		return nil
	})
	c.closed = make(chan struct{})
	c.pending = make(chan *request, 100)
	c.results = map[uint64]chan *response{}
	c.conn = w
	go c.readLoop()
	go c.writeLoop(pings)
	return nil
}

func (c *Client) HasError() bool {
	c.mu.RLock()
	hasErr := c.err != nil
	c.mu.RUnlock()
	return hasErr
}

// FindTransaction corresponds to the Tendermint TxSearch call. It returns the
// set of matching transactions and the total number of results as part of the
// TransactionList response.
//
// The page parameter, which is 1-indexed, can be used to get the specific page
// of results that you're interested in. And the perPage variable defines the
// number of maximum results (up to 100) that you want per page.
//
// And if prove is set to true, then the response will include proofs of the
// transactions' inclusion in the Tendermint blockchain.
func (c *Client) FindTransaction(ctx context.Context, query *Query, page int, perPage int, prove bool) (*TransactionList, error) {
	qs, err := query.Expression()
	if err != nil {
		return nil, err
	}
	resp := &TransactionList{}
	err = c.call(ctx, "tx_search", opts{"query": qs, "page": page, "per_page": perPage, "prove": prove}, resp)
	return resp, err
}

// Genesis corresponds to the Tendermint Genesis call. It returns the
// initial conditions of the Tendermint blockchain.
func (c *Client) Genesis(ctx context.Context) (*Genesis, error) {
	type Container struct {
		Info *Genesis `json:"genesis"`
	}
	resp := &Container{}
	err := c.call(ctx, "genesis", nil, resp)
	return resp.Info, err
}

// GetBlockInfo corresponds to the Tendermint Block call. It returns the
// BlockInfo for the block at the given height, or the latest block if the
// height is 0 or negative.
func (c *Client) GetBlockInfo(ctx context.Context, height int64) (*BlockInfo, error) {
	params := opts{}
	if height > 0 {
		params["height"] = height
	}
	resp := &BlockInfo{}
	err := c.call(ctx, "block", params, resp)
	return resp, err
}

// GetBlockMetas corresponds to the Tendermint Blockchain call. It returns the
// BlockMeta info in descending order for all the blocks within the given
// [minHeight, maxHeight] range. At most 20 BlockMetas will be returned.
//
// If minHeight is less than 0 or negative, it defaults to 1. And if maxHeight
// is 0, it defaults to the height of the current blockchain, i.e. the latest
// block.
func (c *Client) GetBlockMetas(ctx context.Context, minHeight int64, maxHeight int64) ([]*BlockMeta, error) {
	if minHeight <= 0 {
		minHeight = 1
	}
	resp := []*BlockMeta{}
	err := c.call(ctx, "blockchain", opts{"minHeight": minHeight, "maxHeight": maxHeight}, resp)
	return resp, err
}

// GetCommitInfo corresponds to the Tendermint Commit call. It returns the
// CommitInfo for the block at the given height, or the latest block if the
// height is 0 or negative.
func (c *Client) GetCommitInfo(ctx context.Context, height int64) (*CommitInfo, error) {
	params := opts{}
	if height > 0 {
		params["height"] = height
	}
	resp := &CommitInfo{}
	err := c.call(ctx, "commit", params, resp)
	return resp, err
}

// IsNodeReachable corresponds to the Tendermint Health call. It can be used as
// a ping to test whether the Tendermint node is still up and running.
func (c *Client) IsNodeReachable(ctx context.Context) (bool, error) {
	var resp struct{}
	err := c.call(ctx, "health", nil, resp)
	return err != nil, err
}

// NetInfo corresponds to the Tendermint NetInfo call.
func (c *Client) NetInfo(ctx context.Context) (*NetInfo, error) {
	resp := &NetInfo{}
	err := c.call(ctx, "net_info", nil, resp)
	return resp, err
}

// Status corresponds to the Tendermint Status call. It returns a bunch of
// useful info relating to the current state of the Tendermint node.
func (c *Client) Status(ctx context.Context) (*Status, error) {
	resp := &Status{}
	err := c.call(ctx, "status", nil, resp)
	return resp, err
}

// Transaction corresponds to the Tendermint Tx call. It returns a Transaction
// matching the given transaction hash.
func (c *Client) Transaction(ctx context.Context, hash []byte, prove bool) (*Transaction, error) {
	resp := &Transaction{}
	err := c.call(ctx, "tx", opts{"hash": hash, "prove": prove}, resp)
	return resp, err
}

// UnconfirmedTransactions corresponds to the Tendermint UnconfirmedTxs call. It
// returns a list of transaction data for unconfirmed transactions up to the
// given limit. If the given limit is less than 1 or greater than 100, then the
// number of returned transactions defaults to 30.
func (c *Client) UnconfirmedTransactions(ctx context.Context, limit int) ([][]byte, error) {
	type Unconfirmed struct {
		Count        int      `json:"n_txs"`
		Transactions [][]byte `json:"txs"`
	}
	resp := &Unconfirmed{}
	err := c.call(ctx, "unconfirmed_txs", opts{"limit": limit}, resp)
	if err != nil {
		return nil, err
	}
	return resp.Transactions, nil
}

// UnconfirmedTransactionsCount corresponds to the Tendermint NumUnconfirmedTxs
// call. It returns the number of unconfirmed transactions.
func (c *Client) UnconfirmedTransactionsCount(ctx context.Context) (int, error) {
	type Unconfirmed struct {
		Count int `json:"n_txs"`
	}
	resp := &Unconfirmed{}
	err := c.call(ctx, "num_unconfirmed_txs", nil, resp)
	if err != nil {
		return 0, err
	}
	return resp.Count, nil
}

// Validators corresponds to the Tendermint Validators call. It returns the
// validator set at the given height, or the latest block if the height is 0 or
// negative.
func (c *Client) Validators(ctx context.Context, height int) (*ValidatorSet, error) {
	params := opts{}
	if height > 0 {
		params["height"] = height
	}
	resp := &ValidatorSet{}
	err := c.call(ctx, "validators", params, resp)
	return resp, err
}

// The call method encodes the given JSON-RPC 2.0 call over the underlying
// WebSocket connection.
func (c *Client) call(ctx context.Context, method string, params opts, resp interface{}) error {
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	id := c.nextID()
	req := &request{
		ID:      id,
		JSONRPC: "2.0",
		Method:  method,
		Params:  body,
	}
	// At this top-level, we select on either putting the request into the
	// pending write queue, waiting for the provided Context to be done
	// (cancelled, deadline exceeded, etc.), or for the underlying connection to
	// be closed.
	select {
	case c.pending <- req:
		log.Infof("Made %s call\n", method)

		ch := make(chan *response, 1)
		c.mu.Lock()
		c.results[id] = ch
		c.mu.Unlock()
		// Once the request has been put onto the write queue, we select on
		// either receiving the response, or the underlying connection being
		// closed.
		select {
		case resp := <-ch:
			c.mu.Lock()
			delete(c.results, id)
			c.mu.Unlock()
			if resp.Error != nil {
				return fmt.Errorf(
					"rpc: got error response from %s call to Tendermint: %s",
					method, resp.Error)
			}
			if resp != nil {
				return json.Unmarshal(resp.Result, resp)
			}
			return nil
		case <-c.closed:
			return ErrClosed
		}
	case <-c.closed:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) closeWithError(err error) error {
	if c.connClosed {
		return c.err
	}
	if err == nil {
		err = c.conn.Close()
	} else {
		c.conn.Close()
	}
	c.connClosed = true
	if err == nil {
		err = ErrClosed
	}
	c.err = err
	close(c.closed)
	return err
}

func (c *Client) handleError(err error) {
	log.Errorf("Got error: %s", err)
	c.mu.Lock()
	c.closeWithError(err)
	c.mu.Unlock()
}

func (c *Client) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connClosed
}

func (c *Client) nextID() uint64 {
	c.mu.Lock()
	c.lastID++
	next := c.lastID
	c.mu.Unlock()
	return next
}

func (c *Client) readLoop() {
	for {
		if c.isClosed() {
			return
		}
		// TODO(tav): Set a read deadline on the connection. The underlying
		// WebSocket implementation seems to treat all errors as fatal, even
		// read timeouts.
		resp := &response{}
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.handleError(err)
			return
		}
		if err = json.Unmarshal(data, resp); err != nil {
			c.handleError(err)
			return
		}
		c.mu.RLock()
		ch, exists := c.results[resp.ID]
		c.mu.RUnlock()
		if !exists {
			log.Infof("Received unexpected response ID: %d", resp.ID)
			c.Close()
			return
		}
		ch <- resp
	}
}

func (c *Client) writeLoop(pings chan string) {
	for {
		if c.isClosed() {
			return
		}
		select {
		case m := <-pings:
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout)); err != nil {
				c.handleError(err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PongMessage, []byte(m)); err != nil {
				c.handleError(err)
				return
			}
		case req := <-c.pending:
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout)); err != nil {
				c.handleError(err)
				return
			}
			if err := c.conn.WriteJSON(req); err != nil {
				c.handleError(err)
				return
			}
		case <-c.closed:
			return
		}
	}
}
