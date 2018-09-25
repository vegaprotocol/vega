package blockchain

// VEGA Blockchain command definitions, lighter-weight than proto
const (
	CreateOrderCommand    Command = 0x40
	CancelOrderCommand    Command = 0x41
	AmendmentOrderCommand Command = 0x42
)

type Command byte

func (cmd Command) String() string {
	names := [...]string{"Create Order", "Cancel Order", "Amend Order"}
	return names[cmd]
}