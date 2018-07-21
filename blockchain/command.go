package blockchain

// VEGA Blockchain command definitions, lighter-weight than proto
const (
	CreateOrderCommand Command = 0x40
	CancelOrderCommand Command = 0x41
)

type Command byte

func (cmd Command) String() string {
	names := [...]string{"Create Order", "Cancel Order"}
	return names[cmd]
}