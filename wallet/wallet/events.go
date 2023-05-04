package wallet

type EventType string

const (
	// WalletCreatedEventType is raised when a wallet has been created.
	WalletCreatedEventType EventType = "WALLET_CREATED"

	// UnlockedWalletUpdatedEventType is raised when a wallet, that is already
	// unlocked, has been updated.
	UnlockedWalletUpdatedEventType = "UNLOCKED_WALLET_UPDATED"

	// LockedWalletUpdatedEventType is raised when a locked wallet has been
	// updated.
	LockedWalletUpdatedEventType = "LOCKED_WALLET_UPDATED"

	// WalletRemovedEventType is raised when a wallet has been removed.
	WalletRemovedEventType = "WALLET_REMOVED"

	// WalletRenamedEventType is raised when a wallet has been renamed.
	WalletRenamedEventType = "WALLET_RENAMED"

	// WalletHasBeenLockedEventType is raised when the wallet has been locked,
	// either by an external passphrase update, or a timeout.
	WalletHasBeenLockedEventType = "WALLET_HAS_BEEN_LOCKED"
)

type Event struct {
	Type EventType `json:"type"`
	Data EventData `json:"data,omitempty"`
}

type EventData interface {
	isEventData()
}

//nolint:revive
type WalletCreatedEventData struct {
	Name string `json:"name"`
}

func (d WalletCreatedEventData) isEventData() {}

func NewWalletCreatedEvent(walletName string) Event {
	return Event{
		Type: WalletCreatedEventType,
		Data: WalletCreatedEventData{
			Name: walletName,
		},
	}
}

type UnlockedWalletUpdatedEventData struct {
	UpdatedWallet Wallet `json:"updateWallet"`
}

func (d UnlockedWalletUpdatedEventData) isEventData() {}

func NewUnlockedWalletUpdatedEvent(w Wallet) Event {
	return Event{
		Type: UnlockedWalletUpdatedEventType,
		Data: UnlockedWalletUpdatedEventData{
			UpdatedWallet: w,
		},
	}
}

type LockedWalletUpdatedEventData struct {
	Name string `json:"name"`
}

func (d LockedWalletUpdatedEventData) isEventData() {}

func NewLockedWalletUpdateEvent(walletName string) Event {
	return Event{
		Type: LockedWalletUpdatedEventType,
		Data: LockedWalletUpdatedEventData{
			Name: walletName,
		},
	}
}

//nolint:revive
type WalletRemovedEventData struct {
	Name string `json:"name"`
}

func (d WalletRemovedEventData) isEventData() {}

func NewWalletRemovedEvent(walletName string) Event {
	return Event{
		Type: WalletRemovedEventType,
		Data: WalletRemovedEventData{
			Name: walletName,
		},
	}
}

//nolint:revive
type WalletRenamedEventData struct {
	PreviousName string `json:"previousName"`
	NewName      string `json:"newName"`
}

func (d WalletRenamedEventData) isEventData() {}

func NewWalletRenamedEvent(previousWalletName, newWalletName string) Event {
	return Event{
		Type: WalletRenamedEventType,
		Data: WalletRenamedEventData{
			PreviousName: previousWalletName,
			NewName:      newWalletName,
		},
	}
}

//nolint:revive
type WalletHasBeenLockedEventData struct {
	Name string `json:"name"`
}

func (d WalletHasBeenLockedEventData) isEventData() {}

func NewWalletHasBeenLockedEvent(name string) Event {
	return Event{
		Type: WalletHasBeenLockedEventType,
		Data: WalletHasBeenLockedEventData{
			Name: name,
		},
	}
}
