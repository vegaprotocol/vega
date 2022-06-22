// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
