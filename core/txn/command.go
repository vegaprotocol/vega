// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	// RotateKeySubmissionCommand ...
	RotateKeySubmissionCommand Command = 0x50
	// StateVariableProposalCommand ...
	StateVariableProposalCommand Command = 0x51
	// TransferFundsCommand ...
	TransferFundsCommand Command = 0x54
	// CancelTransferFundsCommand ...
	CancelTransferFundsCommand Command = 0x55
	// ValidatorHeartbeatCommand ...
	ValidatorHeartbeatCommand Command = 0x56
	// RotateEthereumKeySubmissionCommand ...
	RotateEthereumKeySubmissionCommand Command = 0x57
	// ProtocolUpgradeCommand Command ...
	ProtocolUpgradeCommand Command = 0x58
	// IssueSignatures Command ...
	IssueSignatures Command = 0x59
	// BatchMarketInstructions Command ...
	BatchMarketInstructions Command = 0x5A
	// StopOrdersSubmissionCommand ...
	StopOrdersSubmissionCommand Command = 0x5B
	// StopOrdersCancellationCommand ...
	StopOrdersCancellationCommand Command = 0x5C
	// CreateReferralSetCommand ...
	CreateReferralSetCommand Command = 0x5D
	// UpdateReferralSetCommand ...
	UpdateReferralSetCommand Command = 0x5E
	// ApplyReferralCodeCommand ...
	ApplyReferralCodeCommand Command = 0x5F
)

var commandName = map[Command]string{
	SubmitOrderCommand:                 "Submit Order",
	CancelOrderCommand:                 "Cancel Order",
	AmendOrderCommand:                  "Amend Order",
	WithdrawCommand:                    "Withdraw",
	ProposeCommand:                     "Proposal",
	VoteCommand:                        "Vote on Proposal",
	AnnounceNodeCommand:                "Register New Node",
	NodeVoteCommand:                    "Node Vote",
	NodeSignatureCommand:               "Node Signature",
	LiquidityProvisionCommand:          "Liquidity Provision Order",
	CancelLiquidityProvisionCommand:    "Cancel Liquidity Provision Order",
	AmendLiquidityProvisionCommand:     "Amend Liquidity Provision Order",
	ChainEventCommand:                  "Chain Event",
	SubmitOracleDataCommand:            "Submit Oracle Data",
	DelegateCommand:                    "Delegate",
	UndelegateCommand:                  "Undelegate",
	RotateKeySubmissionCommand:         "Key Rotate Submission",
	StateVariableProposalCommand:       "State Variable Proposal",
	TransferFundsCommand:               "Transfer Funds",
	CancelTransferFundsCommand:         "Cancel Transfer Funds",
	ValidatorHeartbeatCommand:          "Validator Heartbeat",
	RotateEthereumKeySubmissionCommand: "Ethereum Key Rotate Submission",
	ProtocolUpgradeCommand:             "Protocol Upgrade",
	IssueSignatures:                    "Issue Signatures",
	BatchMarketInstructions:            "Batch Market Instructions",
	StopOrdersSubmissionCommand:        "Stop Orders Submission",
	StopOrdersCancellationCommand:      "Stop Orders Cancellation",
	CreateReferralSetCommand:           "Create Referral Set",
	UpdateReferralSetCommand:           "Update Referral Set",
	ApplyReferralCodeCommand:           "Apply Referral Code",
}

func (cmd Command) IsValidatorCommand() bool {
	switch cmd {
	case NodeSignatureCommand, ChainEventCommand, NodeVoteCommand, ValidatorHeartbeatCommand, RotateKeySubmissionCommand, StateVariableProposalCommand, RotateEthereumKeySubmissionCommand:
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
