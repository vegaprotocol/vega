package txn

// Command ...
type Command byte

// Custom blockchain command encoding, lighter-weight than proto.
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
	// AnnounceNodeCommand ...
	AnnounceNodeCommand Command = 0x47
	// NodeVoteCommand ...
	NodeVoteCommand Command = 0x48
	// NodeSignatureCommand ...
	NodeSignatureCommand Command = 0x49
	// LiquidityProvisionCommand ...
	LiquidityProvisionCommand Command = 0x4A
	// CancelLiquidityProvisionCommand ...
	CancelLiquidityProvisionCommand Command = 0x52
	// AmendLiquidityProvisionCommand ...
	AmendLiquidityProvisionCommand Command = 0x53
	// ChainEventCommand ...
	ChainEventCommand Command = 0x4B
	// SubmitOracleDataCommand ...
	SubmitOracleDataCommand Command = 0x4C
	// DelegateCommand ...
	DelegateCommand Command = 0x4D
	// UndelegateCommand ...
	UndelegateCommand Command = 0x4E
	// CheckpointRestoreCommand ...
	CheckpointRestoreCommand Command = 0x4F
	// KeyRotateSubmissionCommand ...
	KeyRotateSubmissionCommand Command = 0x50
	// StateVariableProposalCommand ...
	StateVariableProposalCommand Command = 0x51
	// TransferFundsCommand ...
	TransferFundsCommand Command = 0x54
	// CancelTransferFundsCommand ...
	CancelTransferFundsCommand Command = 0x55
	// ValidatorHeartbeat ...
	ValidatorHeartbeatCommand Command = 0x56
)

var commandName = map[Command]string{
	SubmitOrderCommand:              "Submit Order",
	CancelOrderCommand:              "Cancel Order",
	AmendOrderCommand:               "Amend Order",
	WithdrawCommand:                 "Withdraw",
	ProposeCommand:                  "Proposal",
	VoteCommand:                     "Vote on Proposal",
	AnnounceNodeCommand:             "Register new Node",
	NodeVoteCommand:                 "Node Vote",
	NodeSignatureCommand:            "Node Signature",
	LiquidityProvisionCommand:       "Liquidity Provision Order",
	CancelLiquidityProvisionCommand: "Cancel LiquidityProvision Order",
	AmendLiquidityProvisionCommand:  "Amend LiquidityProvision Order",
	ChainEventCommand:               "Chain Event",
	SubmitOracleDataCommand:         "Submit Oracle Data",
	DelegateCommand:                 "Delegate",
	UndelegateCommand:               "Undelegate",
	CheckpointRestoreCommand:        "Checkpoint Restore",
	KeyRotateSubmissionCommand:      "Key Rotate Submission",
	StateVariableProposalCommand:    "State Variable Proposal",
	TransferFundsCommand:            "Transfer Funds",
	CancelTransferFundsCommand:      "Cancel Transfer Funds",
	ValidatorHeartbeatCommand:       "Validator Heartbeat",
}

func (cmd Command) IsValidatorCommand() bool {
	switch cmd {
	case CheckpointRestoreCommand, NodeSignatureCommand, ChainEventCommand, NodeVoteCommand, ValidatorHeartbeatCommand, KeyRotateSubmissionCommand, StateVariableProposalCommand:
		return true
	default:
		return false
	}
}

// String return the.
func (cmd Command) String() string {
	s, ok := commandName[cmd]
	if ok {
		return s
	}
	return ""
}
