package blockchain

// CommandKind
type CommandKind byte

const (
	CommandKindSigned   CommandKind = 0x10
	CommandKindUnsigned CommandKind = 0x11
)

// Command ...
type Command byte

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
	// WithdrawCommand ...
	WithdrawCommand Command = 0x44
	// ProposeCommand ...
	ProposeCommand Command = 0x45
	// VoteCommand
	VoteCommand Command = 0x46
	// RegisterNodecommand ...
	RegisterNodeCommand Command = 0x47
	// NodeVoteCommand...
	NodeVoteCommand Command = 0x48
	// NodeSignatureCommand..
	NodeSignatureCommand Command = 0x49
	// ChainEventCommand..
	ChainEventCommand Command = 0x50
)

var commandName = map[Command]string{
	SubmitOrderCommand:         "Submit Order",
	CancelOrderCommand:         "Cancel Order",
	AmendOrderCommand:          "Amend Order",
	NotifyTraderAccountCommand: "Notify Trader Account",
	WithdrawCommand:            "Withdraw",
	ProposeCommand:             "Proposal",
	VoteCommand:                "Vote on Proposal",
	RegisterNodeCommand:        "Register new Node",
	NodeVoteCommand:            "Node Vote",
	NodeSignatureCommand:       "Node Signature",
	ChainEventCommand:          "Chain Event",
}

var commandKindName = map[CommandKind]string{
	CommandKindSigned:   "SignedTx",
	CommandKindUnsigned: "UnsignedTx",
}

// String return the
func (cmd Command) String() string {
	s, ok := commandName[cmd]
	if ok {
		return s
	}
	return ""
}

func (cmd CommandKind) String() string {
	s, ok := commandKindName[cmd]
	if ok {
		return s
	}
	return ""
}
