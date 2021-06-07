package processor

import (
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/crypto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/txn"
	"github.com/golang/protobuf/proto"
)

type TxV2 struct {
	originalTx []byte
	tx         *commandspb.Transaction
	inputData  *commandspb.InputData
}

func DecodeTxV2(payload []byte) (*TxV2, error) {
	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal(payload, tx); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction: %w", err)
	}

	inputData := &commandspb.InputData{}
	if err := proto.Unmarshal(tx.InputData, inputData); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction from signed bundle: %w", err)
	}

	return &TxV2{
		originalTx: payload,
		tx:         tx,
		inputData:  inputData,
	}, nil
}

func (t TxV2) Command() txn.Command {
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
	case *commandspb.InputData_ProposalSubmission:
		return txn.ProposeCommand
	case *commandspb.InputData_NodeRegistration:
		return txn.RegisterNodeCommand
	case *commandspb.InputData_NodeVote:
		return txn.NodeVoteCommand
	case *commandspb.InputData_NodeSignature:
		return txn.NodeSignatureCommand
	case *commandspb.InputData_ChainEvent:
		return txn.ChainEventCommand
	case *commandspb.InputData_OracleDataSubmission:
		return txn.SubmitOracleDataCommand
	default:
		panic("unsupported command")
	}
}

func (t TxV2) Unmarshal(i interface{}) error {
	switch cmd := t.inputData.Command.(type) {
	case *commandspb.InputData_OrderSubmission:
		if underlyingCmd, ok := i.(*commandspb.OrderSubmission); ok {
			*underlyingCmd = *cmd.OrderSubmission
			return nil
		} else {
			panic("failed to unmarshall to OrderSubmission")
		}
	case *commandspb.InputData_OrderCancellation:
		if underlyingCmd, ok := i.(*commandspb.OrderCancellation); ok {
			*underlyingCmd = *cmd.OrderCancellation
			return nil
		} else {
			panic("failed to unmarshall to OrderCancellation")
		}
	case *commandspb.InputData_OrderAmendment:
		if underlyingCmd, ok := i.(*commandspb.OrderAmendment); ok {
			*underlyingCmd = *cmd.OrderAmendment
			return nil
		} else {
			panic("failed to unmarshall to OrderAmendment")
		}
	case *commandspb.InputData_VoteSubmission:
		if underlyingCmd, ok := i.(*commandspb.VoteSubmission); ok {
			*underlyingCmd = *cmd.VoteSubmission
			return nil
		} else {
			panic("failed to unmarshall to VoteSubmission")
		}
	case *commandspb.InputData_WithdrawSubmission:
		if underlyingCmd, ok := i.(*commandspb.WithdrawSubmission); ok {
			*underlyingCmd = *cmd.WithdrawSubmission
			return nil
		} else {
			panic("failed to unmarshall to WithdrawSubmission")
		}
	case *commandspb.InputData_LiquidityProvisionSubmission:
		if underlyingCmd, ok := i.(*commandspb.LiquidityProvisionSubmission); ok {
			*underlyingCmd = *cmd.LiquidityProvisionSubmission
			return nil
		} else {
			panic("failed to unmarshall to LiquidityProvisionSubmission")
		}
	case *commandspb.InputData_ProposalSubmission:
		if underlyingCmd, ok := i.(*commandspb.ProposalSubmission); ok {
			*underlyingCmd = *cmd.ProposalSubmission
			return nil
		} else {
			panic("failed to unmarshall to ProposalSubmission")
		}
	case *commandspb.InputData_NodeRegistration:
		if underlyingCmd, ok := i.(*commandspb.NodeRegistration); ok {
			*underlyingCmd = *cmd.NodeRegistration
			return nil
		} else {
			panic("failed to unmarshall to NodeRegistration")
		}
	case *commandspb.InputData_NodeVote:
		if underlyingCmd, ok := i.(*commandspb.NodeVote); ok {
			*underlyingCmd = *cmd.NodeVote
			return nil
		} else {
			panic("failed to unmarshall to NodeVote")
		}
	case *commandspb.InputData_NodeSignature:
		if underlyingCmd, ok := i.(*commandspb.NodeSignature); ok {
			*underlyingCmd = *cmd.NodeSignature
			return nil
		} else {
			panic("failed to unmarshall to NodeSignature")
		}
	case *commandspb.InputData_ChainEvent:
		if underlyingCmd, ok := i.(*commandspb.ChainEvent); ok {
			*underlyingCmd = *cmd.ChainEvent
			return nil
		} else {
			panic("failed to unmarshall to ChainEvent")
		}
	case *commandspb.InputData_OracleDataSubmission:
		if underlyingCmd, ok := i.(*commandspb.OracleDataSubmission); ok {
			*underlyingCmd = *cmd.OracleDataSubmission
			return nil
		} else {
			panic("failed to unmarshall to OracleDataSubmission")
		}
	default:
		panic("unsupported command")
	}
}

func (t TxV2) PubKey() []byte {
	decodedPubKey, err := hex.DecodeString(t.tx.GetPubKey())
	if err != nil {
		panic("pub key should be hex encoded")
	}
	return decodedPubKey
}

func (t TxV2) Party() string {
	return hex.EncodeToString([]byte(t.tx.GetPubKey()))
}

func (t TxV2) Hash() []byte {
	return crypto.Hash(t.originalTx)
}

func (t TxV2) Signature() []byte {
	decodedSig, err := hex.DecodeString(t.tx.Signature.Value)
	if err != nil {
		panic("signature should be hex encoded")
	}
	return decodedSig
}

func (t TxV2) Validate() error {
	return commands.CheckTransaction(t.tx)
}

func (t TxV2) BlockHeight() uint64 {
	return t.inputData.BlockHeight
}
