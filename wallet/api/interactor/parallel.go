package interactor

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/wallet/api"
)

var ErrTooManyRequests = errors.New("there are too many requests")

// ParallelInteractor is built to handle multiple requests at a time.
type ParallelInteractor struct {
	// userCtx is the context used to listen to the user-side cancellation
	// requests. It interrupts the wait on responses.
	userCtx context.Context

	outboundCh chan<- Interaction
}

func (i *ParallelInteractor) NotifyInteractionSessionBegan(_ context.Context, traceID string, workflow api.WorkflowType, numberOfSteps uint8) error {
	interaction := Interaction{
		TraceID: traceID,
		Name:    InteractionSessionBeganName,
		Data: InteractionSessionBegan{
			Workflow:             string(workflow),
			MaximumNumberOfSteps: numberOfSteps,
		},
	}

	select {
	case i.outboundCh <- interaction:
		return nil
	default:
		return ErrTooManyRequests
	}
}

func (i *ParallelInteractor) NotifyInteractionSessionEnded(_ context.Context, traceID string) {
	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    InteractionSessionEndedName,
		Data:    InteractionSessionEnded{},
	}
}

func (i *ParallelInteractor) NotifyError(ctx context.Context, traceID string, t api.ErrorType, err error) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    ErrorOccurredName,
		Data: ErrorOccurred{
			Type:  string(t),
			Error: err.Error(),
		},
	}
}

func (i *ParallelInteractor) NotifySuccessfulTransaction(ctx context.Context, traceID string, stepNumber uint8, txHash, deserializedInputData, tx string, sentAt time.Time, host string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    TransactionSucceededName,
		Data: TransactionSucceeded{
			DeserializedInputData: deserializedInputData,
			TxHash:                txHash,
			Tx:                    tx,
			SentAt:                sentAt,
			Node: SelectedNode{
				Host: host,
			},
			StepNumber: stepNumber,
		},
	}
}

func (i *ParallelInteractor) NotifyFailedTransaction(ctx context.Context, traceID string, stepNumber uint8, deserializedInputData, tx string, err error, sentAt time.Time, host string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.outboundCh <- Interaction{
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
			StepNumber: stepNumber,
		},
	}
}

func (i *ParallelInteractor) NotifySuccessfulRequest(ctx context.Context, traceID string, stepNumber uint8, message string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestSucceededName,
		Data: RequestSucceeded{
			Message:    message,
			StepNumber: stepNumber,
		},
	}
}

func (i *ParallelInteractor) Log(ctx context.Context, traceID string, t api.LogType, msg string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    LogName,
		Data: Log{
			Type:    string(t),
			Message: msg,
		},
	}
}

func (i *ParallelInteractor) RequestWalletConnectionReview(ctx context.Context, traceID string, stepNumber uint8, hostname string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", api.ErrRequestInterrupted
	}

	responseCh := make(chan Interaction, 1)
	defer close(responseCh)

	controlCh := make(chan error, 1)
	defer close(controlCh)

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestWalletConnectionReviewName,
		Data: RequestWalletConnectionReview{
			Hostname:   hostname,
			StepNumber: stepNumber,
			ResponseCh: responseCh,
			ControlCh:  controlCh,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, WalletConnectionDecisionName, responseCh, controlCh)
	if err != nil {
		return "", err
	}

	decision, ok := interaction.Data.(WalletConnectionDecision)
	if !ok {
		return "", InvalidResponsePayloadError(WalletConnectionDecisionName)
	}

	return decision.ConnectionApproval, nil
}

func (i *ParallelInteractor) RequestWalletSelection(ctx context.Context, traceID string, stepNumber uint8, hostname string, availableWallets []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", api.ErrRequestInterrupted
	}

	responseCh := make(chan Interaction, 1)
	defer close(responseCh)

	controlCh := make(chan error, 1)
	defer close(controlCh)

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestWalletSelectionName,
		Data: RequestWalletSelection{
			Hostname:         hostname,
			AvailableWallets: availableWallets,
			StepNumber:       stepNumber,
			ResponseCh:       responseCh,
			ControlCh:        controlCh,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, SelectedWalletName, responseCh, controlCh)
	if err != nil {
		return "", err
	}

	selectedWallet, ok := interaction.Data.(SelectedWallet)
	if !ok {
		return "", InvalidResponsePayloadError(SelectedWalletName)
	}

	return selectedWallet.Wallet, nil
}

func (i *ParallelInteractor) RequestPassphrase(ctx context.Context, traceID string, stepNumber uint8, wallet, reason string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", api.ErrRequestInterrupted
	}

	responseCh := make(chan Interaction, 1)
	defer close(responseCh)

	controlCh := make(chan error, 1)
	defer close(controlCh)

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestPassphraseName,
		Data: RequestPassphrase{
			Wallet:     wallet,
			Reason:     reason,
			StepNumber: stepNumber,
			ResponseCh: responseCh,
			ControlCh:  controlCh,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, EnteredPassphraseName, responseCh, controlCh)
	if err != nil {
		return "", err
	}

	enteredPassphrase, ok := interaction.Data.(EnteredPassphrase)
	if !ok {
		return "", InvalidResponsePayloadError(EnteredPassphraseName)
	}
	return enteredPassphrase.Passphrase, nil
}

func (i *ParallelInteractor) RequestPermissionsReview(ctx context.Context, traceID string, stepNumber uint8, hostname, wallet string, perms map[string]string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	responseCh := make(chan Interaction, 1)
	defer close(responseCh)

	controlCh := make(chan error, 1)
	defer close(controlCh)

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestPermissionsReviewName,
		Data: RequestPermissionsReview{
			Hostname:    hostname,
			Wallet:      wallet,
			Permissions: perms,
			StepNumber:  stepNumber,
			ResponseCh:  responseCh,
			ControlCh:   controlCh,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, DecisionName, responseCh, controlCh)
	if err != nil {
		return false, err
	}

	approval, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}
	return approval.Approved, nil
}

func (i *ParallelInteractor) RequestTransactionReviewForSending(ctx context.Context, traceID string, stepNumber uint8, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	responseCh := make(chan Interaction, 1)
	defer close(responseCh)

	controlCh := make(chan error, 1)
	defer close(controlCh)

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestTransactionReviewForSendingName,
		Data: RequestTransactionReviewForSending{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
			StepNumber:  stepNumber,
			ResponseCh:  responseCh,
			ControlCh:   controlCh,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, DecisionName, responseCh, controlCh)
	if err != nil {
		return false, err
	}

	approval, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}
	return approval.Approved, nil
}

func (i *ParallelInteractor) RequestTransactionReviewForSigning(ctx context.Context, traceID string, stepNumber uint8, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	responseCh := make(chan Interaction, 1)
	defer close(responseCh)

	controlCh := make(chan error, 1)
	defer close(controlCh)

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestTransactionReviewForSigningName,
		Data: RequestTransactionReviewForSigning{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
			StepNumber:  stepNumber,
			ResponseCh:  responseCh,
			ControlCh:   controlCh,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, DecisionName, responseCh, controlCh)
	if err != nil {
		return false, err
	}

	approval, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}
	return approval.Approved, nil
}

func (i *ParallelInteractor) RequestTransactionReviewForChecking(ctx context.Context, traceID string, stepNumber uint8, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	responseCh := make(chan Interaction, 1)
	defer close(responseCh)

	controlCh := make(chan error, 1)
	defer close(controlCh)

	i.outboundCh <- Interaction{
		TraceID: traceID,
		Name:    RequestTransactionReviewForCheckingName,
		Data: RequestTransactionReviewForChecking{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
			StepNumber:  stepNumber,
			ResponseCh:  responseCh,
			ControlCh:   controlCh,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, DecisionName, responseCh, controlCh)
	if err != nil {
		return false, err
	}

	approval, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}
	return approval.Approved, nil
}

func (i *ParallelInteractor) waitForResponse(ctx context.Context, traceID string, expectedResponseName InteractionName, responseCh <-chan Interaction, controlCh chan<- error) (Interaction, error) {
	var response Interaction
	running := true
	for running {
		select {
		case <-ctx.Done():
			controlCh <- api.ErrRequestInterrupted
			return Interaction{}, api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return Interaction{}, api.ErrUserCloseTheConnection
		case r := <-responseCh:
			response = r
			running = false
		}
	}

	if response.TraceID != traceID {
		return Interaction{}, TraceIDMismatchError(traceID, response.TraceID)
	}

	if response.Name == CancelRequestName {
		return Interaction{}, api.ErrUserCancelledTheRequest
	}

	if response.Name != expectedResponseName {
		return Interaction{}, WrongResponseTypeError(expectedResponseName, response.Name)
	}

	return response, nil
}

func NewParallelInteractor(userCtx context.Context, outboundCh chan<- Interaction) *ParallelInteractor {
	i := &ParallelInteractor{
		userCtx:    userCtx,
		outboundCh: outboundCh,
	}

	return i
}
