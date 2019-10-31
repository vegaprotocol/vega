package blockchain

// Custom blockchain command encoding, lighter-weight than proto
const (
	// SubmitOrderCommand ...
	SubmitOrderCommand Command = 0x40
	// CancelOrderCommand ..
	CancelOrderCommand Command = 0x41
	// AmendOrderCommand ...
	AmendOrderCommand Command = 0x42
	// NotifyTraderAccountCommand ...
	NotifyTraderAccountCommand Command = 0x43
	// WithdrawCommant ...
	WithdrawCommand = 0x44
)

// Command ...
type Command byte

var commandName = map[Command]string{
	SubmitOrderCommand:         "Submit Order",
	CancelOrderCommand:         "Cancel Order",
	AmendOrderCommand:          "Amend Order",
	NotifyTraderAccountCommand: "Notify Trader Account",
	WithdrawCommand:            "Withdraw",
}

// String return the
func (cmd Command) String() string {
	s, ok := commandName[cmd]
	if ok {
		return s
	}
	return ""
}
