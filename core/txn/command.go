// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	// UpdateMarginModeCommand ...
	UpdateMarginModeCommand Command = 0x60
	// JoinTeamCommand ...
	JoinTeamCommand Command = 0x61
	// BatchProposeCommand ...
	BatchProposeCommand Command = 0x62
	// UpdatePartyProfileCommand ...
	UpdatePartyProfileCommand Command = 0x63
	// SubmitAMMCommand ...
	SubmitAMMCommand Command = 0x64
	// AmendAMMCommand ...
	AmendAMMCommand Command = 0x65
	// CancelAMMCommand ...
	CancelAMMCommand Command = 0x66
	// DelayedTransactionsWrapper ...
	DelayedTransactionsWrapper Command = 0x67
	// CreateVaultCommand ...
	CreateVaultCommand Command = 0x68
	// UpdateVaultCommand ...
	UpdateVaultCommand Command = 0x69
	// CloseVaultCommand ...
	CloseVaultCommand Command = 0x6A
	// DepositToVaultCommand ...
	DepositToVaultCommand Command = 0x6B
	// WithdrawFromVaultCommand ...
	WithdrawFromVaultCommand Command = 0x6C
	// ChangeVaultOwnershipCommand ...
	ChangeVaultOwnershipCommand Command = 0x6D
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
	UpdateMarginModeCommand:            "Update Margin Mode",
	JoinTeamCommand:                    "Join Team",
	BatchProposeCommand:                "Batch Proposal",
	UpdatePartyProfileCommand:          "Update Party Profile",
	SubmitAMMCommand:                   "Submit AMM",
	AmendAMMCommand:                    "Amend AMM",
	CancelAMMCommand:                   "Cancel AMM",
	DelayedTransactionsWrapper:         "Delayed Transactions Wrapper",
	CreateVaultCommand:                 "Create Vault",
	UpdateVaultCommand:                 "Update Vault",
	CloseVaultCommand:                  "Close Vault",
	DepositToVaultCommand:              "Deposit To Vault",
	WithdrawFromVaultCommand:           "Withdraw From Vault",
	ChangeVaultOwnershipCommand:        "Change Vault Ownership",
}

func (cmd Command) IsValidatorCommand() bool {
	switch cmd {
	case DelayedTransactionsWrapper, NodeSignatureCommand, ChainEventCommand, NodeVoteCommand, ValidatorHeartbeatCommand, RotateKeySubmissionCommand, StateVariableProposalCommand, RotateEthereumKeySubmissionCommand:
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
