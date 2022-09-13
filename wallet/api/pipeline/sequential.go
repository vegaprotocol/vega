package pipeline

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/wallet/api"
)

var (
	ErrTraceIDMismatch   = errors.New("the trace IDs between request and response mismatch")
	ErrWrongResponseType = errors.New("the received response does not match the expected response type")
)

// SequentialPipeline is a pipeline built to handle one request at a time.
// Concurrent requests are not supported and will result in errors.
type SequentialPipeline struct {
	// userCtx is the context used to listen to the user-side cancellation
	// requests. It interrupts the wait on responses.
	userCtx context.Context

	receptionChan chan<- Envelope
	responseChan  <-chan Envelope
}

func (s *SequentialPipeline) NotifyError(ctx context.Context, traceID string, t api.ErrorType, err error) {
	if err := ctx.Err(); err != nil {
		return
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: ErrorOccurred{
			Type:  string(t),
			Error: err.Error(),
		},
	}
}

func (s *SequentialPipeline) NotifySuccessfulRequest(ctx context.Context, traceID string) {
	if err := ctx.Err(); err != nil {
		return
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: RequestSucceeded{},
	}
}

func (s *SequentialPipeline) RequestWalletConnectionReview(ctx context.Context, traceID, hostname string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: RequestWalletConnectionReview{
			Hostname: hostname,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return false, api.ErrRequestInterrupted
		case <-s.userCtx.Done():
			return false, api.ErrUserCloseTheConnection
		case response := <-s.responseChan:
			if response.TraceID != traceID {
				return false, ErrTraceIDMismatch
			}
			decision, ok := response.Content.(Decision)
			if !ok {
				return false, ErrWrongResponseType
			}
			return decision.Approved, nil
		}
	}
}

func (s *SequentialPipeline) RequestWalletSelection(ctx context.Context, traceID, hostname string, availableWallets []string) (api.SelectedWallet, error) {
	if err := ctx.Err(); err != nil {
		return api.SelectedWallet{}, api.ErrRequestInterrupted
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: RequestWalletSelection{
			Hostname:         hostname,
			AvailableWallets: availableWallets,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return api.SelectedWallet{}, api.ErrRequestInterrupted
		case <-s.userCtx.Done():
			return api.SelectedWallet{}, api.ErrUserCloseTheConnection
		case response := <-s.responseChan:
			if response.TraceID != traceID {
				return api.SelectedWallet{}, ErrTraceIDMismatch
			}
			selectedWallet, ok := response.Content.(api.SelectedWallet)
			if !ok {
				return api.SelectedWallet{}, ErrWrongResponseType
			}
			return selectedWallet, nil
		}
	}
}

func (s *SequentialPipeline) RequestPassphrase(ctx context.Context, traceID, wallet string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", api.ErrRequestInterrupted
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: RequestPassphrase{
			Wallet: wallet,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return "", api.ErrRequestInterrupted
		case <-s.userCtx.Done():
			return "", api.ErrUserCloseTheConnection
		case response := <-s.responseChan:
			if response.TraceID != traceID {
				return "", ErrTraceIDMismatch
			}
			enteredPassphrase, ok := response.Content.(EnteredPassphrase)
			if !ok {
				return "", ErrWrongResponseType
			}
			return enteredPassphrase.Passphrase, nil
		}
	}
}

func (s *SequentialPipeline) RequestPermissionsReview(ctx context.Context, traceID, hostname, wallet string, perms map[string]string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: RequestPermissionsReview{
			Hostname:    hostname,
			Wallet:      wallet,
			Permissions: perms,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return false, api.ErrRequestInterrupted
		case <-s.userCtx.Done():
			return false, api.ErrUserCloseTheConnection
		case response := <-s.responseChan:
			if response.TraceID != traceID {
				return false, ErrTraceIDMismatch
			}
			decision, ok := response.Content.(Decision)
			if !ok {
				return false, ErrWrongResponseType
			}
			return decision.Approved, nil
		}
	}
}

func (s *SequentialPipeline) RequestTransactionSendingReview(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: RequestTransactionSendingReview{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return false, api.ErrRequestInterrupted
		case <-s.userCtx.Done():
			return false, api.ErrUserCloseTheConnection
		case response := <-s.responseChan:
			if response.TraceID != traceID {
				return false, ErrTraceIDMismatch
			}
			approval, ok := response.Content.(Decision)
			if !ok {
				return false, ErrWrongResponseType
			}
			return approval.Approved, nil
		}
	}
}

func (s *SequentialPipeline) RequestTransactionSigningReview(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: RequestTransactionSigningReview{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return false, api.ErrRequestInterrupted
		case <-s.userCtx.Done():
			return false, api.ErrUserCloseTheConnection
		case response := <-s.responseChan:
			if response.TraceID != traceID {
				return false, ErrTraceIDMismatch
			}
			approval, ok := response.Content.(Decision)
			if !ok {
				return false, ErrWrongResponseType
			}
			return approval.Approved, nil
		}
	}
}

func (s *SequentialPipeline) NotifyTransactionStatus(ctx context.Context, traceID, txHash, tx string, err error, sentAt time.Time) {
	if err := ctx.Err(); err != nil {
		return
	}

	s.receptionChan <- Envelope{
		TraceID: traceID,
		Content: TransactionStatus{
			TxHash: txHash,
			Tx:     tx,
			Error:  err,
			SentAt: sentAt,
		},
	}
}

func NewSequentialPipeline(userCtx context.Context, receptionChan chan<- Envelope, responseChan <-chan Envelope) *SequentialPipeline {
	return &SequentialPipeline{
		userCtx:       userCtx,
		receptionChan: receptionChan,
		responseChan:  responseChan,
	}
}

type Envelope struct {
	// TraceID is an identifier specifically made for client front-end to keep
	// track of a transaction during all of its lifetime, from transaction
	// review to sending confirmation and in-memory history.
	// It shouldn't be confused with the transaction hash that get assigned
	// only after it has been sent to the network.
	TraceID string      `json:"traceID"`
	Content interface{} `json:"content"`
}

type ErrorOccurred struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

type RequestWalletConnectionReview struct {
	Hostname string `json:"hostname"`
}

type RequestWalletSelection struct {
	Hostname         string   `json:"hostname"`
	AvailableWallets []string `json:"availableWallets"`
}

type RequestPassphrase struct {
	Wallet string `json:"wallet"`
}

type EnteredPassphrase struct {
	Passphrase string `json:"passphrase"`
}

type RequestPermissionsReview struct {
	Hostname    string            `json:"hostname"`
	Wallet      string            `json:"wallet"`
	Permissions map[string]string `json:"permissions"`
}

type Decision struct {
	Approved bool `json:"approved"`
}

type RequestTransactionSendingReview struct {
	Hostname    string    `json:"hostname"`
	Wallet      string    `json:"wallet"`
	PublicKey   string    `json:"publicKey"`
	Transaction string    `json:"transaction"`
	ReceivedAt  time.Time `json:"receivedAt"`
}

type RequestTransactionSigningReview struct {
	Hostname    string    `json:"hostname"`
	Wallet      string    `json:"wallet"`
	PublicKey   string    `json:"publicKey"`
	Transaction string    `json:"transaction"`
	ReceivedAt  time.Time `json:"receivedAt"`
}

type TransactionStatus struct {
	TxHash string    `json:"txHash"`
	Tx     string    `json:"tx"`
	Error  error     `json:"error"`
	SentAt time.Time `json:"sentAt"`
}

type RequestSucceeded struct{}
