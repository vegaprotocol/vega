// Package rpc implements a WebSocket RPC client interface to Tendermint.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DefaultAddress provides the default host:port for the Tendermint RPC server.
const DefaultAddress = "127.0.0.1:46657"

// Endpoint specifies the WebSockets endpoint path on the Tendermint RPC server.
const Endpoint = "/websocket"

// Errors returned by the Client.
var (
	ErrCheckTxFailed = errors.New("rpc: CheckTx failed during the CreateTransaction call")
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
	Data    string `json:"data,omitempty"`
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
	sync.RWMutex     // Protects the closed, connClosed, err, lastID and results struct fields.
	Address          string
	HandshakeTimeout time.Duration
	WriteTimeout     time.Duration
	closed           chan struct{}
	conn             *websocket.Conn
	connClosed       bool
	err              error
	lastID           uint64
	pending          chan *request
	results          map[uint64]chan *response
}

// CreateTransactionResponse represents the data returned from the
// CreateTransaction call.
type CreateTransactionResponse struct {
	// Code represents the exit code from the corresponding CheckTx ABCI call. A
	// non-zero Code value represents an error. The meaning of non-zero codes is
	// specific to the given ABCI app that is being used.
	Code uint32 `json:"code"`
	Data []byte `json:"data"`
	Hash []byte `json:"hash"`
	Log  string `json:"log"`
}

// CreateTransaction corresponds to the BroadcastTxSync Tendermint call. It adds
// the given transaction data to Tendermint's mempool and returns data from the
// corresponding CheckTx response.
//
// If the given transaction data fails CheckTx, then the method will return a
// ErrCheckTxFailed error, and the caller can inspect the returned
// CreateTransactionResponse value for the application-specific error code and
// log message.
func (c *Client) CreateTransaction(ctx context.Context, txdata []byte) (*CreateTransactionResponse, error) {
	data, err := c.call(ctx, "broadcast_tx_sync", opts{"tx": txdata})
	if err != nil {
		return nil, err
	}
	resp := &CreateTransactionResponse{}
	err = json.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return resp, ErrCheckTxFailed
	}
	return resp, nil
}

// Close terminates the underlying WebSocket connection.
func (c *Client) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.closeWithError(nil)
}

// Connect initialises the Client and establishes a WebSocket connection to the
// Tendermint endpoint. It must only be called once per Client.
func (c *Client) Connect() error {
	dialer := &websocket.Dialer{
		HandshakeTimeout: c.HandshakeTimeout,
	}
	w, _, err := dialer.Dial("ws://"+c.Address+Endpoint, nil)
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
	c.results = make(map[uint64]chan *response)
	c.conn = w
	go c.readLoop()
	go c.writeLoop(pings)
	return nil
}

// IsNodeReachable corresponds to the Tendermint Health call. It can be used as
// a ping to test whether the Tendermint node is still up and running.
func (c *Client) IsNodeReachable(ctx context.Context) (bool, error) {
	resp, err := c.call(ctx, "health", opts{})
	if err != nil {
		return false, err
	}
	return string(resp) == "{}", nil
}

// The call method encodes the given JSON-RPC 2.0 call over the underlying
// WebSocket connection.
func (c *Client) call(ctx context.Context, method string, params opts) (json.RawMessage, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
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
		fmt.Printf(".. Made %s call\n", method)
		ch := make(chan *response, 1)
		c.Lock()
		c.results[id] = ch
		c.Unlock()
		// Once the request has been put onto the write queue, we select on
		// either receiving the response, or the underlying connection being
		// closed.
		select {
		case resp := <-ch:
			c.Lock()
			delete(c.results, id)
			c.Unlock()
			if resp.Error != nil {
				return nil, fmt.Errorf(
					"rpc: got error response from %s call to Tendermint: %s",
					method, resp.Error)
			}
			return resp.Result, nil
		case <-c.closed:
			return nil, ErrClosed
		}
	case <-c.closed:
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
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
	log.Printf("Got error: %s", err)
	c.Lock()
	c.closeWithError(err)
	c.Unlock()
}

func (c *Client) isClosed() bool {
	c.RLock()
	defer c.RUnlock()
	return c.connClosed
}

func (c *Client) nextID() uint64 {
	c.Lock()
	c.lastID++
	next := c.lastID
	c.Unlock()
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
		c.RLock()
		ch, exists := c.results[resp.ID]
		c.RUnlock()
		if !exists {
			// TODO(tav): We probably don't want to actually quit here.
			log.Fatalf("ERROR: received unexpected response ID '%d' for a JSON-RPC call", resp.ID)
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
		}
	}
}
