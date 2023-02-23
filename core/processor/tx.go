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

package processor

import (
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/cometbft/cometbft/crypto/tmhash"
)

type Tx struct {
	originalTx []byte
	tx         *commandspb.Transaction
	inputData  *commandspb.InputData
	pow        *commandspb.ProofOfWork
	version    commandspb.TxVersion
}

func DecodeTxNoValidation(payload []byte) (*Tx, error) {
	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal(payload, tx); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction: %w", err)
	}

	inputData, err := commands.CheckInputData(tx.InputData)
	if err := err.ErrorOrNil(); err != nil {
		return nil, err
	}

	return &Tx{
		originalTx: payload,
		tx:         tx,
		inputData:  inputData,
		pow:        tx.Pow,
		version:    tx.Version,
	}, nil
}

func DecodeTx(payload []byte, chainID string) (*Tx, error) {
	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal(payload, tx); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction: %w", err)
	}

	inputData, err := commands.CheckTransaction(tx, chainID)
	if err != nil {
		return nil, err
	}

	return &Tx{
		originalTx: payload,
		tx:         tx,
		inputData:  inputData,
		pow:        tx.Pow,
		version:    tx.Version,
	}, nil
}

func (t Tx) Command() txn.Command {
	switch t.inputData.Command.(type) {
	case *commandspb.InputData_OrderSubmission:
		return txn.SubmitOrderCommand
	case *commandspb.InputData_OrderCancellation:
		return txn.CancelOrderCommand
	case *commandspb.InputData_OrderAmendment:
		return txn.AmendOrderCommand
	case *commandspb.InputData_VoteSubmission:
		return txn.VoteCommand
	case *commandspb.InputData_WithdrawSubmission:
		return txn.WithdrawCommand
	case *commandspb.InputData_LiquidityProvisionSubmission:
		return txn.LiquidityProvisionCommand
	case *commandspb.InputData_LiquidityProvisionCancellation:
		return txn.CancelLiquidityProvisionCommand
	case *commandspb.InputData_LiquidityProvisionAmendment:
		return txn.AmendLiquidityProvisionCommand
	case *commandspb.InputData_ProposalSubmission:
		return txn.ProposeCommand
	case *commandspb.InputData_AnnounceNode:
		return txn.AnnounceNodeCommand
	case *commandspb.InputData_NodeVote:
		return txn.NodeVoteCommand
	case *commandspb.InputData_NodeSignature:
		return txn.NodeSignatureCommand
	case *commandspb.InputData_ChainEvent:
		return txn.ChainEventCommand
	case *commandspb.InputData_OracleDataSubmission:
		return txn.SubmitOracleDataCommand
	case *commandspb.InputData_DelegateSubmission:
		return txn.DelegateCommand
	case *commandspb.InputData_UndelegateSubmission:
		return txn.UndelegateCommand
	case *commandspb.InputData_KeyRotateSubmission:
		return txn.RotateKeySubmissionCommand
	case *commandspb.InputData_StateVariableProposal:
		return txn.StateVariableProposalCommand
	case *commandspb.InputData_Transfer:
		return txn.TransferFundsCommand
	case *commandspb.InputData_CancelTransfer:
		return txn.CancelTransferFundsCommand
	case *commandspb.InputData_ValidatorHeartbeat:
		return txn.ValidatorHeartbeatCommand
	case *commandspb.InputData_EthereumKeyRotateSubmission:
		return txn.RotateEthereumKeySubmissionCommand
	case *commandspb.InputData_ProtocolUpgradeProposal:
		return txn.ProtocolUpgradeCommand
	case *commandspb.InputData_IssueSignatures:
		return txn.IssueSignatures
	case *commandspb.InputData_BatchMarketInstructions:
		return txn.BatchMarketInstructions
	default:
		panic("unsupported command")
	}
}

func (t Tx) GetPoWNonce() uint64 {
	// The proof-of-work is not required by validator commands. So, it can be
	// nil.
	if t.pow == nil {
		return 0
	}
	return t.pow.Nonce
}

func (t Tx) GetPoWTID() string {
	// The proof-of-work is not required by validator commands. So, it can be
	// nil.
	if t.pow == nil {
		return ""
	}
	return t.pow.Tid
}

func (t Tx) GetVersion() uint32 { return uint32(t.version) }

func (t Tx) GetCmd() interface{} {
	switch cmd := t.inputData.Command.(type) {
	case *commandspb.InputData_OrderSubmission:
		return cmd.OrderSubmission
	case *commandspb.InputData_OrderCancellation:
		return cmd.OrderCancellation
	case *commandspb.InputData_OrderAmendment:
		return cmd.OrderAmendment
	case *commandspb.InputData_VoteSubmission:
		return cmd.VoteSubmission
	case *commandspb.InputData_WithdrawSubmission:
		return cmd.WithdrawSubmission
	case *commandspb.InputData_LiquidityProvisionSubmission:
		return cmd.LiquidityProvisionSubmission
	case *commandspb.InputData_LiquidityProvisionCancellation:
		return cmd.LiquidityProvisionCancellation
	case *commandspb.InputData_LiquidityProvisionAmendment:
		return cmd.LiquidityProvisionAmendment
	case *commandspb.InputData_ProposalSubmission:
		return cmd.ProposalSubmission
	case *commandspb.InputData_AnnounceNode:
		return cmd.AnnounceNode
	case *commandspb.InputData_NodeVote:
		return cmd.NodeVote
	case *commandspb.InputData_NodeSignature:
		return cmd.NodeSignature
	case *commandspb.InputData_ChainEvent:
		return cmd.ChainEvent
	case *commandspb.InputData_OracleDataSubmission:
		return cmd.OracleDataSubmission
	case *commandspb.InputData_DelegateSubmission:
		return cmd.DelegateSubmission
	case *commandspb.InputData_UndelegateSubmission:
		return cmd.UndelegateSubmission
	case *commandspb.InputData_KeyRotateSubmission:
		return cmd.KeyRotateSubmission
	case *commandspb.InputData_StateVariableProposal:
		return cmd.StateVariableProposal
	case *commandspb.InputData_Transfer:
		return cmd.Transfer
	case *commandspb.InputData_CancelTransfer:
		return cmd.CancelTransfer
	case *commandspb.InputData_ValidatorHeartbeat:
		return cmd.ValidatorHeartbeat
	case *commandspb.InputData_EthereumKeyRotateSubmission:
		return cmd.EthereumKeyRotateSubmission
	case *commandspb.InputData_ProtocolUpgradeProposal:
		return cmd.ProtocolUpgradeProposal
	case *commandspb.InputData_IssueSignatures:
		return cmd.IssueSignatures
	case *commandspb.InputData_BatchMarketInstructions:
		return cmd.BatchMarketInstructions
	default:
		return errors.New("unsupported command")
	}
}

func (t Tx) Unmarshal(i interface{}) error {
	switch cmd := t.inputData.Command.(type) {
	case *commandspb.InputData_ProtocolUpgradeProposal:
		underlyingCmd, ok := i.(*commandspb.ProtocolUpgradeProposal)
		if !ok {
			return errors.New("failed to unmarshall to ProtocolUpgradeProposal")
		}
		*underlyingCmd = *cmd.ProtocolUpgradeProposal
	case *commandspb.InputData_OrderSubmission:
		underlyingCmd, ok := i.(*commandspb.OrderSubmission)
		if !ok {
			return errors.New("failed to unmarshall to OrderSubmission")
		}
		*underlyingCmd = *cmd.OrderSubmission
	case *commandspb.InputData_OrderCancellation:
		underlyingCmd, ok := i.(*commandspb.OrderCancellation)
		if !ok {
			return errors.New("failed to unmarshall to OrderCancellation")
		}
		*underlyingCmd = *cmd.OrderCancellation
	case *commandspb.InputData_OrderAmendment:
		underlyingCmd, ok := i.(*commandspb.OrderAmendment)
		if !ok {
			return errors.New("failed to unmarshall to OrderAmendment")
		}
		*underlyingCmd = *cmd.OrderAmendment
	case *commandspb.InputData_VoteSubmission:
		underlyingCmd, ok := i.(*commandspb.VoteSubmission)
		if !ok {
			return errors.New("failed to unmarshall to VoteSubmission")
		}
		*underlyingCmd = *cmd.VoteSubmission
	case *commandspb.InputData_WithdrawSubmission:
		underlyingCmd, ok := i.(*commandspb.WithdrawSubmission)
		if !ok {
			return errors.New("failed to unmarshall to WithdrawSubmission")
		}
		*underlyingCmd = *cmd.WithdrawSubmission
	case *commandspb.InputData_LiquidityProvisionSubmission:
		underlyingCmd, ok := i.(*commandspb.LiquidityProvisionSubmission)
		if !ok {
			return errors.New("failed to unmarshall to LiquidityProvisionSubmission")
		}
		*underlyingCmd = *cmd.LiquidityProvisionSubmission
	case *commandspb.InputData_LiquidityProvisionCancellation:
		underlyingCmd, ok := i.(*commandspb.LiquidityProvisionCancellation)
		if !ok {
			return errors.New("failed to unmarshall to LiquidityProvisionCancellation")
		}
		*underlyingCmd = *cmd.LiquidityProvisionCancellation
	case *commandspb.InputData_LiquidityProvisionAmendment:
		underlyingCmd, ok := i.(*commandspb.LiquidityProvisionAmendment)
		if !ok {
			return errors.New("failed to unmarshall to LiquidityProvisionAmendment")
		}
		*underlyingCmd = *cmd.LiquidityProvisionAmendment
	case *commandspb.InputData_ProposalSubmission:
		underlyingCmd, ok := i.(*commandspb.ProposalSubmission)
		if !ok {
			return errors.New("failed to unmarshall to ProposalSubmission")
		}
		*underlyingCmd = *cmd.ProposalSubmission
	case *commandspb.InputData_AnnounceNode:
		underlyingCmd, ok := i.(*commandspb.AnnounceNode)
		if !ok {
			return errors.New("failed to unmarshall to AnnounceNode")
		}
		*underlyingCmd = *cmd.AnnounceNode
	case *commandspb.InputData_NodeVote:
		underlyingCmd, ok := i.(*commandspb.NodeVote)
		if !ok {
			return errors.New("failed to unmarshall to NodeVote")
		}
		*underlyingCmd = *cmd.NodeVote
	case *commandspb.InputData_NodeSignature:
		underlyingCmd, ok := i.(*commandspb.NodeSignature)
		if !ok {
			return errors.New("failed to unmarshall to NodeSignature")
		}
		*underlyingCmd = *cmd.NodeSignature
	case *commandspb.InputData_ChainEvent:
		underlyingCmd, ok := i.(*commandspb.ChainEvent)
		if !ok {
			return errors.New("failed to unmarshall to ChainEvent")
		}
		*underlyingCmd = *cmd.ChainEvent
	case *commandspb.InputData_OracleDataSubmission:
		underlyingCmd, ok := i.(*commandspb.OracleDataSubmission)
		if !ok {
			return errors.New("failed to unmarshall to OracleDataSubmission")
		}
		*underlyingCmd = *cmd.OracleDataSubmission
	case *commandspb.InputData_DelegateSubmission:
		underlyingCmd, ok := i.(*commandspb.DelegateSubmission)
		if !ok {
			return errors.New("failed to unmarshall to DelegateSubmission")
		}
		*underlyingCmd = *cmd.DelegateSubmission
	case *commandspb.InputData_UndelegateSubmission:
		underlyingCmd, ok := i.(*commandspb.UndelegateSubmission)
		if !ok {
			return errors.New("failed to unmarshall to UndelegateSubmission")
		}
		*underlyingCmd = *cmd.UndelegateSubmission
	case *commandspb.InputData_KeyRotateSubmission:
		underlyingCmd, ok := i.(*commandspb.KeyRotateSubmission)
		if !ok {
			return errors.New("failed to unmarshal KeyRotateSubmission")
		}
		*underlyingCmd = *cmd.KeyRotateSubmission
	case *commandspb.InputData_StateVariableProposal:
		underlyingCmd, ok := i.(*commandspb.StateVariableProposal)
		if !ok {
			return errors.New("failed to unmarshal StateVariableProposal")
		}
		*underlyingCmd = *cmd.StateVariableProposal
	case *commandspb.InputData_Transfer:
		underlyingCmd, ok := i.(*commandspb.Transfer)
		if !ok {
			return errors.New("failed to unmarshal TransferFunds")
		}
		*underlyingCmd = *cmd.Transfer
	case *commandspb.InputData_CancelTransfer:
		underlyingCmd, ok := i.(*commandspb.CancelTransfer)
		if !ok {
			return errors.New("failed to unmarshal CancelTransferFunds")
		}
		*underlyingCmd = *cmd.CancelTransfer
	case *commandspb.InputData_ValidatorHeartbeat:
		underlyingCmd, ok := i.(*commandspb.ValidatorHeartbeat)
		if !ok {
			return errors.New("failed to unmarshal ValidatorHeartbeat")
		}
		*underlyingCmd = *cmd.ValidatorHeartbeat
	case *commandspb.InputData_EthereumKeyRotateSubmission:
		underlyingCmd, ok := i.(*commandspb.EthereumKeyRotateSubmission)
		if !ok {
			return errors.New("failed to unmarshal EthereumKeyRotateSubmission")
		}
		*underlyingCmd = *cmd.EthereumKeyRotateSubmission
	case *commandspb.InputData_IssueSignatures:
		underlyingCmd, ok := i.(*commandspb.IssueSignatures)
		if !ok {
			return errors.New("failed to unmarshall to IssueSignatures")
		}
		*underlyingCmd = *cmd.IssueSignatures
	case *commandspb.InputData_BatchMarketInstructions:
		underlyingCmd, ok := i.(*commandspb.BatchMarketInstructions)
		if !ok {
			return errors.New("failed to unmarshall to BatchMarketInstructions")
		}
		*underlyingCmd = *cmd.BatchMarketInstructions
	default:
		return errors.New("unsupported command")
	}
	return nil
}

func (t Tx) PubKey() []byte {
	decodedPubKey, err := hex.DecodeString(t.tx.GetPubKey())
	if err != nil {
		panic("pub key should be hex encoded")
	}
	return decodedPubKey
}

func (t Tx) PubKeyHex() string {
	return t.tx.GetPubKey()
}

func (t Tx) Party() string {
	return t.tx.GetPubKey()
}

func (t Tx) Hash() []byte {
	return tmhash.Sum(t.originalTx)
}

func (t Tx) Signature() []byte {
	decodedSig, err := hex.DecodeString(t.tx.Signature.Value)
	if err != nil {
		panic("signature should be hex encoded")
	}
	return decodedSig
}

func (t Tx) BlockHeight() uint64 {
	return t.inputData.BlockHeight
}
