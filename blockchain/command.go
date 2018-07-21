package blockchain

// VEGA Blockchain command definitions, lighter-weight than proto
const (
	CreateOrder Command = 0x40
	CancelOrder Command = 0x41
)

type Command byte

func (cmd Command) String() string {
	names := [...]string{"Create Order", "Cancel Order"}
	return names[cmd]
}
