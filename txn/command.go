package txn

// Command ...
type Command byte

// Custom blockchain command encoding, lighter-weight than proto
const (
	// SubmitOrderCommand ...
	SubmitOrderCommand Command = 0x40
	// CancelOrderCommand ...
	CancelOrderCommand Command = 0x41
	// AmendOrderCommand ...
	AmendOrderCommand Command = 0x42
	// WithdrawCommand ...
	WithdrawCommand Command = 0x44
	// ProposeCommand ...
	ProposeCommand Command = 0x45
	// VoteCommand ...
	VoteCommand Command = 0x46
	// RegisterNodeCommand ...
	RegisterNodeCommand Command = 0x47
	// NodeVoteCommand ...
	NodeVoteCommand Command = 0x48
	// NodeSignatureCommand ...
	NodeSignatureCommand Command = 0x49
	// LiquidityProvisionCommand ...
	LiquidityProvisionCommand Command = 0x4A
	// ChainEventCommand ...
	ChainEventCommand Command = 0x50
	// SubmitOracleDataCommand ...
	SubmitOracleDataCommand Command = 0x51
)

var commandName = map[Command]string{
	SubmitOrderCommand:        "Submit Order",
	CancelOrderCommand:        "Cancel Order",
	AmendOrderCommand:         "Amend Order",
	WithdrawCommand:           "Withdraw",
	ProposeCommand:            "Proposal",
	VoteCommand:               "Vote on Proposal",
	RegisterNodeCommand:       "Register new Node",
	NodeVoteCommand:           "Node Vote",
	NodeSignatureCommand:      "Node Signature",
	LiquidityProvisionCommand: "Liquidity Provision Order",
	ChainEventCommand:         "Chain Event",
	SubmitOracleDataCommand:   "Submit Oracle Data",
}

// String return the
func (cmd Command) String() string {
	s, ok := commandName[cmd]
	if ok {
		return s
	}
	return ""
}
