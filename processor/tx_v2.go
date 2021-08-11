package processor

import (
	"encoding/hex"
	"errors"
	"fmt"

	wcrypto "code.vegaprotocol.io/go-wallet/crypto"
	"code.vegaprotocol.io/protos/commands"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
)

var (
	ErrUnsupportedFromValueInTransaction = errors.New("unsupported value from `from` field in transaction")
)

type TxV2 struct {
	originalTx []byte
	tx         *commandspb.Transaction
	inputData  *commandspb.InputData
	err        error
}

func DecodeTxV2(payload []byte) (*TxV2, error) {
	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal(payload, tx); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction: %w", err)
	}

	inputData, err := commands.CheckTransaction(tx)
	if err != nil {
		return nil, err
	}

	err = checkSignature(tx)
	if err != nil {
		return nil, err
	}

	return &TxV2{
		originalTx: payload,
		tx:         tx,
		inputData:  inputData,
		err:        err,
	}, nil
}

func checkSignature(tx *commandspb.Transaction) error {
	algo, err := wcrypto.NewSignatureAlgorithm(tx.Signature.Algo, tx.Signature.Version)
	if err != nil {
		return err
	}

	decodedSig, err := hex.DecodeString(tx.Signature.Value)
	if err != nil {
		return err
	}

	if len(tx.GetPubKey()) == 0 {
		return ErrUnsupportedFromValueInTransaction
	} 
	pubKeyOrAddress, err := hex.DecodeString(tx.GetPubKey())
	if err != nil {
		return fmt.Errorf("invalid public key, %w", err)
	}

	verified, err := algo.Verify(pubKeyOrAddress, tx.InputData, decodedSig)
	if err != nil {
		return err
	}

	if !verified {
		return ErrInvalidSignature
	}

	return nil
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
	case *commandspb.InputData_DelegateSubmission:
		return txn.DelegateCommand
	case *commandspb.InputData_UndelegateSubmission:
		return txn.UndelegateCommand
	default:
		panic("unsupported command")
	}
}

func (t TxV2) Unmarshal(i interface{}) error {
	switch cmd := t.inputData.Command.(type) {
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
	case *commandspb.InputData_ProposalSubmission:
		underlyingCmd, ok := i.(*commandspb.ProposalSubmission)
		if !ok {
			return errors.New("failed to unmarshall to ProposalSubmission")
		}
		*underlyingCmd = *cmd.ProposalSubmission
	case *commandspb.InputData_NodeRegistration:
		underlyingCmd, ok := i.(*commandspb.NodeRegistration)
		if !ok {
			return errors.New("failed to unmarshall to NodeRegistration")
		}
		*underlyingCmd = *cmd.NodeRegistration
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

	default:
		return errors.New("unsupported command")
	}
	return nil
}

func (t TxV2) PubKey() []byte {
	decodedPubKey, err := hex.DecodeString(t.tx.GetPubKey())
	if err != nil {
		panic("pub key should be hex encoded")
	}
	return decodedPubKey
}

func (t TxV2) Party() string {
	return t.tx.GetPubKey()
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
	return t.err
}

func (t TxV2) BlockHeight() uint64 {
	return t.inputData.BlockHeight
}
