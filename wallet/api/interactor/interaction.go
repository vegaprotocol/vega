package interactor

import (
	"encoding/json"
	"time"

	"github.com/mitchellh/mapstructure"
)

const (
	CancelRequestName                       InteractionName = "CANCEL_REQUEST"
	DecisionName                            InteractionName = "DECISION"
	EnteredPassphraseName                   InteractionName = "ENTERED_PASSPHRASE"
	ErrorOccurredName                       InteractionName = "ERROR_OCCURRED"
	InteractionSessionBeganName             InteractionName = "INTERACTION_SESSION_BEGAN"
	InteractionSessionEndedName             InteractionName = "INTERACTION_SESSION_ENDED"
	LogName                                 InteractionName = "LOG"
	RequestPassphraseName                   InteractionName = "REQUEST_PASSPHRASE"
	RequestPermissionsReviewName            InteractionName = "REQUEST_PERMISSIONS_REVIEW"
	RequestSucceededName                    InteractionName = "REQUEST_SUCCEEDED"
	RequestTransactionReviewForSendingName  InteractionName = "REQUEST_TRANSACTION_REVIEW_FOR_SENDING"
	RequestTransactionReviewForSigningName  InteractionName = "REQUEST_TRANSACTION_REVIEW_FOR_SIGNING"
	RequestTransactionReviewForCheckingName InteractionName = "REQUEST_TRANSACTION_REVIEW_FOR_CHECKING"
	RequestWalletConnectionReviewName       InteractionName = "REQUEST_WALLET_CONNECTION_REVIEW"
	RequestWalletSelectionName              InteractionName = "REQUEST_WALLET_SELECTION"
	SelectedWalletName                      InteractionName = "SELECTED_WALLET"
	TransactionFailedName                   InteractionName = "TRANSACTION_FAILED"
	TransactionSucceededName                InteractionName = "TRANSACTION_SUCCEEDED"
	WalletConnectionDecisionName            InteractionName = "WALLET_CONNECTION_DECISION"
)

type InteractionName string

// Interaction wraps the messages the JSON-RPC API sends to the wallet front-end
// along with information about the context.
type Interaction struct {
	// TraceID is an identifier specifically made for client front-end to keep
	// track of a transaction during all of its lifetime, from transaction
	// review to sending confirmation and in-memory history.
	// It shouldn't be confused with the transaction hash that is the
	// transaction identifier.
	TraceID string `json:"traceID"`

	// Name is the name of the interaction. This helps to figure out how the
	// data payload should be parsed.
	Name InteractionName `json:"name"`

	// Data is the generic field that hold the data of the specific interaction.
	Data interface{} `json:"data"`
}

func (f *Interaction) UnmarshalJSON(data []byte) error {
	input := &struct {
		TraceID string                 `json:"traceID"`
		Name    InteractionName        `json:"name"`
		Data    map[string]interface{} `json:"data"`
	}{}

	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	f.TraceID = input.TraceID
	f.Name = input.Name

	switch input.Name {
	case CancelRequestName:
		data := CancelRequest{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestWalletConnectionReviewName:
		data := RequestWalletConnectionReview{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestWalletSelectionName:
		data := RequestWalletSelection{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestPassphraseName:
		data := RequestPassphrase{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestPermissionsReviewName:
		data := RequestPermissionsReview{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestTransactionReviewForSendingName:
		data := RequestTransactionReviewForSending{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestTransactionReviewForSigningName:
		data := RequestTransactionReviewForSigning{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestTransactionReviewForCheckingName:
		data := RequestTransactionReviewForChecking{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case WalletConnectionDecisionName:
		data := WalletConnectionDecision{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case DecisionName:
		data := Decision{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case EnteredPassphraseName:
		data := EnteredPassphrase{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case SelectedWalletName:
		data := SelectedWallet{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case InteractionSessionBeganName:
		data := InteractionSessionBegan{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case InteractionSessionEndedName:
		data := InteractionSessionEnded{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case RequestSucceededName:
		data := RequestSucceeded{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case ErrorOccurredName:
		data := ErrorOccurred{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case TransactionSucceededName:
		data := TransactionSucceeded{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case TransactionFailedName:
		data := TransactionFailed{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	case LogName:
		data := Log{}
		if err := mapstructure.Decode(input.Data, &data); err != nil {
			return err
		}
		f.Data = data
	}

	return nil
}

// CancelRequest cancels a request that is waiting for user inputs.
// It can't cancel a request when there is no response awaited.
type CancelRequest struct{}

// RequestWalletConnectionReview is a request emitted when a third-party
// application wants to connect to a wallet.
type RequestWalletConnectionReview struct {
	Hostname string `json:"hostname"`

	StepNumber uint8 `json:"stepNumber"`
}

// RequestWalletSelection is a request emitted when the service requires the user
// to select a wallet. It is emitted after the user approved the wallet connection
// from a third-party application.
// It should be answered by an interactor.SelectedWallet response.
type RequestWalletSelection struct {
	Hostname         string   `json:"hostname"`
	AvailableWallets []string `json:"availableWallets"`

	StepNumber uint8 `json:"stepNumber"`
}

// RequestPassphrase is a request emitted when the service wants to confirm
// the user has access to the wallet.
// It should be answered by an interactor.EnteredPassphrase response.
type RequestPassphrase struct {
	Wallet string `json:"wallet"`
	Reason string `json:"reason"`

	StepNumber uint8 `json:"stepNumber"`
}

// RequestPermissionsReview is a review request emitted when a third-party
// application performs an operation that requires an update to the permissions.
type RequestPermissionsReview struct {
	Hostname    string            `json:"hostname"`
	Wallet      string            `json:"wallet"`
	Permissions map[string]string `json:"permissions"`

	StepNumber uint8 `json:"stepNumber"`
}

// RequestTransactionReviewForSending is a review request emitted when a third-party
// application wants to send a transaction.
type RequestTransactionReviewForSending struct {
	Hostname    string    `json:"hostname"`
	Wallet      string    `json:"wallet"`
	PublicKey   string    `json:"publicKey"`
	Transaction string    `json:"transaction"`
	ReceivedAt  time.Time `json:"receivedAt"`

	StepNumber uint8 `json:"stepNumber"`
}

// RequestTransactionReviewForChecking is a review request when a third-party
// application wants the user to check a transaction.
type RequestTransactionReviewForChecking struct {
	Hostname    string    `json:"hostname"`
	Wallet      string    `json:"wallet"`
	PublicKey   string    `json:"publicKey"`
	Transaction string    `json:"transaction"`
	ReceivedAt  time.Time `json:"receivedAt"`

	StepNumber uint8 `json:"stepNumber"`
}

// RequestTransactionReviewForSigning is a review request when a third-party
// application wants to sign a transaction.
type RequestTransactionReviewForSigning struct {
	Hostname    string    `json:"hostname"`
	Wallet      string    `json:"wallet"`
	PublicKey   string    `json:"publicKey"`
	Transaction string    `json:"transaction"`
	ReceivedAt  time.Time `json:"receivedAt"`

	StepNumber uint8 `json:"stepNumber"`
}

// WalletConnectionDecision is a specific response for interactor.RequestWalletConnectionReview.
type WalletConnectionDecision struct {
	// ConnectionApproval tells if the third-party application is authorized
	// to connect to a wallet.
	// The value is the string representation of a preferences.ConnectionApproval.
	ConnectionApproval string `json:"connectionApproval"`
}

// Decision is a generic response for the following review requests:
//   - RequestPermissionsReview
//   - RequestTransactionReviewForSigning
//   - RequestTransactionReviewForSending
type Decision struct {
	// Approved is set to true if the request is accepted by the user, false
	// otherwise.
	Approved bool `json:"approved"`
}

// EnteredPassphrase contains the passphrase of a given wallet the user
// entered. It's a response to the interactor.RequestPassphrase.
type EnteredPassphrase struct {
	Passphrase string `json:"passphrase"`
}

// SelectedWallet contains the wallet chosen by the user for a given action.
type SelectedWallet struct {
	Wallet string `json:"wallet"`
}

// InteractionSessionBegan is a notification that is emitted when the interaction
// session begin. It only carries informational value on a request lifecycle. This
// is the first notification to be emitted and is always emitted when a request
// comes in.
type InteractionSessionBegan struct {
	Workflow             string `json:"workflow"`
	MaximumNumberOfSteps uint8  `json:"maximumNumberOfSteps"`
}

// InteractionSessionEnded is a notification that is emitted when the interaction
// session ended. This is the last notification to be emitted and is always emitted,
// regardless of the request status. It only carries informational value on a
// request lifecycle. The success or failure status of a request is carried by
// the interactor.RequestSucceeded and interactor.ErrorOccurred notifications,
// respectively.
// Nothing should be expected after receiving this notification.
type InteractionSessionEnded struct{}

// ErrorOccurred is a generic notification emitted when the something failed.
// This notification can wrap an internal failure as much as a user input error.
// Receiving this notification doesn't necessarily mean the overall
// request failed. The request should be considered as failed when this notification
// is followed by the interactor.InteractionSessionEnded notification.
type ErrorOccurred struct {
	// Type is an enumeration that gives information about the origin of the error.
	// The value is the string representation of an api.ErrorType.
	Type string `json:"type"`

	// Error is the error message describing the reason of the failure.
	Error string `json:"error"`
}

// RequestSucceeded is a generic notification emitted when the request succeeded,
// meaning no error has been encountered.
// This notification is emitted only once.
type RequestSucceeded struct {
	// Message can contain a custom success message.
	Message string `json:"message"`

	StepNumber uint8 `json:"stepNumber"`
}

// TransactionSucceeded is a notification sent when the sending of a
// transaction succeeded. It replaces the RequestSucceeded notification as it
// carries specific information that wallet front-ends may use for history.
// This notification is emitted only once.
type TransactionSucceeded struct {
	// TxHash is the hash of the transaction that is used to uniquely identify
	// a transaction. It can be used to retrieve a transaction in the explorer.
	TxHash string `json:"txHash"`

	// DeserializedInputData is the input data bundled in the transaction in a
	// human-readable format.
	DeserializedInputData string `json:"deserializedInputData"`

	// Tx is the true representation of the transaction that is sent to the
	// network.
	Tx string `json:"tx"`

	// SentAt is the time a which the transaction has been sent to the network.
	// It's useful to build a list of the sending in a chronological order on
	// the front-ends.
	SentAt time.Time `json:"sentAt"`

	// Node contains all the information related to the node selected for the
	// sending of the transaction.
	Node SelectedNode `json:"node"`

	StepNumber uint8 `json:"stepNumber"`
}

// TransactionFailed is a notification sent when the sending of a
// transaction failed for any reason. It replaces the ErrorOccurred notification
// as it carries specific information that wallet front-ends may use for
// investigation.
// This notification is emitted only once.
type TransactionFailed struct {
	// DeserializedInputData is the input data bundled in the transaction in a
	// human-readable format.
	DeserializedInputData string `json:"deserializedInputData"`

	// Tx is the true representation of the transaction that is sent to the
	// network.
	Tx string `json:"tx"`

	// Error is the error message describing the reason of the failure.
	Error error `json:"error"`

	// SentAt is the time a which the transaction has been sent to the network.
	// It's useful to build a list of the sending in a chronological order on
	// the front-ends.
	SentAt time.Time `json:"sentAt"`

	// Node contains all the information related to the node selected for the
	// sending of the transaction.
	Node SelectedNode `json:"node"`

	StepNumber uint8 `json:"stepNumber"`
}

type SelectedNode struct {
	Host string `json:"host"`
}

// Log is a generic event that shouldn't be confused with a notification. A log
// is conceptually different. Whatever the type (error, warning...), a log is just
// an information about the internal processing that we think is worth to
// broadcast to wallet front-ends. That said, it can be safely ignored if not
// needed. That is where is differs from the notifications.
type Log struct {
	// Type is an enumeration that gives information about the level of log.
	// The value is the string representation of an api.LogType.
	Type string `json:"type"`

	// Message is the log message itself.
	Message string `json:"message"`
}
