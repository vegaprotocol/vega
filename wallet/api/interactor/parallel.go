package interactor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/wallet/api"
)

type ParallelInteractor struct {
	// userCtx is the context used to listen to the user-side cancellation
	// requests. It interrupts the wait on responses.
	userCtx context.Context

	startSessionChan chan<- *OngoingSession

	mu       sync.Mutex
	sessions map[string]*OngoingSession
}

func (i *ParallelInteractor) NotifyInteractionSessionBegan(ctx context.Context, traceID string) error {
	if err := i.ensureCanProceed(ctx); err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	_, ok := i.sessions[traceID]
	if ok {
		panic(fmt.Errorf("trace ID %q is already used in a session", traceID))
	}

	userReceptionChan := make(chan Interaction, 1)
	userResponseChan := make(chan Interaction, 1)
	i.sessions[traceID] = &OngoingSession{
		userReceptionChan: userReceptionChan,
		userResponseChan:  userResponseChan,
	}

	userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    InteractionSessionBeganName,
		Data:    InteractionSessionBegan{},
	}

	return nil
}

func (i *ParallelInteractor) NotifyInteractionSessionEnded(ctx context.Context, traceID string) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session, ok := i.sessions[traceID]
	if !ok {
		return
	}

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    InteractionSessionEndedName,
		Data:    InteractionSessionEnded{},
	}

	close(session.userReceptionChan)
	close(session.userResponseChan)

	delete(i.sessions, traceID)
}

func (i *ParallelInteractor) NotifyError(ctx context.Context, traceID string, t api.ErrorType, err error) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	i.mustGetSession(traceID).userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    ErrorOccurredName,
		Data: ErrorOccurred{
			Type:  string(t),
			Error: err.Error(),
		},
	}
}

func (i *ParallelInteractor) NotifySuccessfulRequest(ctx context.Context, traceID string, message string) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	i.mustGetSession(traceID).userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestSucceededName,
		Data: RequestSucceeded{
			Message: message,
		},
	}
}

func (i *ParallelInteractor) NotifySuccessfulTransaction(ctx context.Context, traceID, txHash, deserializedInputData, tx string, sentAt time.Time, host string) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    TransactionSucceededName,
		Data: TransactionSucceeded{
			DeserializedInputData: deserializedInputData,
			Tx:                    tx,
			TxHash:                txHash,
			SentAt:                sentAt,
			Node: SelectedNode{
				Host: host,
			},
		},
	}
}

func (i *ParallelInteractor) NotifyFailedTransaction(ctx context.Context, traceID, deserializedInputData, tx string, err error, sentAt time.Time, host string) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    TransactionFailedName,
		Data: TransactionFailed{
			DeserializedInputData: deserializedInputData,
			Tx:                    tx,
			Error:                 err,
			SentAt:                sentAt,
			Node: SelectedNode{
				Host: host,
			},
		},
	}
}

func (i *ParallelInteractor) Log(ctx context.Context, traceID string, t api.LogType, msg string) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	i.mustGetSession(traceID).userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    LogName,
		Data: Log{
			Type:    string(t),
			Message: msg,
		},
	}
}

func (i *ParallelInteractor) RequestWalletConnectionReview(ctx context.Context, traceID, hostname string) (string, error) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return "", err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestWalletConnectionReviewName,
		Data: RequestWalletConnectionReview{
			Hostname: hostname,
		},
	}

	interaction, err := i.waitForResponse(ctx, session.userResponseChan, traceID, WalletConnectionDecisionName)
	if err != nil {
		return "", err
	}

	decision, ok := interaction.Data.(WalletConnectionDecision)
	if !ok {
		return "", InvalidResponsePayloadError(WalletConnectionDecisionName)
	}

	return decision.ConnectionApproval, nil
}

func (i *ParallelInteractor) RequestWalletSelection(ctx context.Context, traceID, hostname string, availableWallets []string) (api.SelectedWallet, error) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return api.SelectedWallet{}, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestWalletSelectionName,
		Data: RequestWalletSelection{
			Hostname:         hostname,
			AvailableWallets: availableWallets,
		},
	}

	interaction, err := i.waitForResponse(ctx, session.userResponseChan, traceID, SelectedWalletName)
	if err != nil {
		return api.SelectedWallet{}, err
	}

	selectedWallet, ok := interaction.Data.(SelectedWallet)
	if !ok {
		return api.SelectedWallet{}, InvalidResponsePayloadError(SelectedWalletName)
	}

	return api.SelectedWallet{
		Wallet:     selectedWallet.Wallet,
		Passphrase: selectedWallet.Wallet,
	}, nil
}

func (i *ParallelInteractor) RequestPassphrase(ctx context.Context, traceID, wallet string) (string, error) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return "", err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestPassphraseName,
		Data: RequestPassphrase{
			Wallet: wallet,
		},
	}

	interaction, err := i.waitForResponse(ctx, session.userResponseChan, traceID, EnteredPassphraseName)
	if err != nil {
		return "", err
	}

	enteredPassphrase, ok := interaction.Data.(EnteredPassphrase)
	if !ok {
		return "", InvalidResponsePayloadError(EnteredPassphraseName)
	}

	return enteredPassphrase.Passphrase, nil
}

func (i *ParallelInteractor) RequestPermissionsReview(ctx context.Context, traceID, hostname, wallet string, perms map[string]string) (bool, error) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return false, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestPermissionsReviewName,
		Data: RequestPermissionsReview{
			Hostname:    hostname,
			Wallet:      wallet,
			Permissions: perms,
		},
	}

	interaction, err := i.waitForResponse(ctx, session.userResponseChan, traceID, DecisionName)
	if err != nil {
		return false, err
	}

	decision, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}

	return decision.Approved, nil
}

func (i *ParallelInteractor) RequestTransactionReviewForSending(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return false, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestTransactionReviewForSendingName,
		Data: RequestTransactionReviewForSending{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
		},
	}

	interaction, err := i.waitForResponse(ctx, session.userResponseChan, traceID, DecisionName)
	if err != nil {
		return false, err
	}

	decision, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}

	return decision.Approved, nil
}

func (i *ParallelInteractor) RequestTransactionReviewForSigning(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := i.ensureCanProceed(ctx); err != nil {
		return false, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	session := i.mustGetSession(traceID)

	session.userReceptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestTransactionReviewForSigningName,
		Data: RequestTransactionReviewForSigning{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
		},
	}

	interaction, err := i.waitForResponse(ctx, session.userResponseChan, traceID, DecisionName)
	if err != nil {
		return false, err
	}

	decision, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}

	return decision.Approved, nil
}

func (i *ParallelInteractor) mustGetSession(traceID string) *OngoingSession {
	session, ok := i.sessions[traceID]
	if !ok {
		panic(fmt.Errorf("no session for trace ID %q, you need to start a session first", traceID))
	}

	return session
}

func (i *ParallelInteractor) ensureCanProceed(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return api.ErrRequestInterrupted
	case <-i.userCtx.Done():
		return api.ErrUserCloseTheConnection
	default:
		break
	}
	return nil
}

func (i *ParallelInteractor) waitForResponse(ctx context.Context, userResponseChan <-chan Interaction, traceID string, name InteractionName) (Interaction, error) {
	var response Interaction
	running := true
	for running {
		select {
		case <-ctx.Done():
			return Interaction{}, api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return Interaction{}, api.ErrUserCloseTheConnection
		case r := <-userResponseChan:
			response = r
			running = false
		}
	}

	if response.TraceID != traceID {
		return Interaction{}, TraceIDMismatchError(traceID, response.TraceID)
	}

	if response.Name == CancelRequestName {
		return Interaction{}, api.ErrUserCanceledTheRequest
	}

	if response.Name != name {
		return Interaction{}, WrongResponseTypeError(name, response.Name)
	}

	return response, nil
}

func NewParallelInteractor(userCtx context.Context, startSessionChan chan<- *OngoingSession) *ParallelInteractor {
	return &ParallelInteractor{
		userCtx:          userCtx,
		startSessionChan: startSessionChan,
		sessions:         map[string]*OngoingSession{},
	}
}

type OngoingSession struct {
	userReceptionChan chan Interaction
	userResponseChan  chan Interaction
}

func (s *OngoingSession) Receive() Interaction {
	return <-s.userReceptionChan
}

func (s *OngoingSession) Send(interaction Interaction) {
	s.userReceptionChan <- interaction
}
