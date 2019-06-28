package blockchain

// Custom blockchain command encoding, lighter-weight than proto
const (
	SubmitOrderCommand         Command = 0x40
	CancelOrderCommand         Command = 0x41
	AmendOrderCommand          Command = 0x42
	NotifyTraderAccountCommand Command = 0x43
)

type Command byte

func (cmd Command) String() string {
	names := [...]string{"Submit Order", "Cancel Order", "Amend Order", "Notify Trader Account"}
	return names[cmd]
}
