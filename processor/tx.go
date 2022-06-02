package processor

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/crypto/tmhash"

	"code.vegaprotocol.io/protos/commands"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/txn"
	wcrypto "code.vegaprotocol.io/vegawallet/crypto"

	"code.vegaprotocol.io/vega/libs/proto"
)

var ErrUnsupportedFromValueInTransaction = errors.New("unsupported value from `from` field in transaction")

type Tx struct {
	originalTx []byte
	tx         *commandspb.Transaction
	inputData  *commandspb.InputData
	err        error
	pow        *commandspb.ProofOfWork
	version    uint32
}

func DecodeTxNoValidation(payload []byte) (*Tx, error) {
	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal(payload, tx); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction: %w", err)
	}
	input := commandspb.InputData{}
	if err := proto.Unmarshal(tx.InputData, &input); err != nil {
		return nil, fmt.Errorf("unable to unmarshal input data: %w", err)
	}

	return &Tx{
		originalTx: payload,
		tx:         tx,
		inputData:  &input,
		err:        nil,
		pow:        tx.Pow,
		version:    tx.Version,
	}, nil
}

func DecodeTx(payload []byte) (*Tx, error) {
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

	return &Tx{
		originalTx: payload,
		tx:         tx,
		inputData:  inputData,
		err:        err,
		pow:        tx.Pow,
		version:    tx.Version,
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
	case *commandspb.InputData_RestoreSnapshotSubmission:
		return txn.CheckpointRestoreCommand
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
	default:
		panic("unsupported command")
	}
}

func (t Tx) GetPoWNonce() uint64 {
	if t.version > 1 && t.pow != nil {
		return t.pow.Nonce
	}
	return 0
}

func (t Tx) GetPoWTID() string {
	if t.version > 1 && t.pow != nil {
		return t.pow.Tid
	}
	return ""
}

func (t Tx) GetVersion() uint32 { return t.version }

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
	case *commandspb.InputData_RestoreSnapshotSubmission:
		return cmd.RestoreSnapshotSubmission
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
	default:
		return errors.New("unsupported command")
	}
}

func (t Tx) Unmarshal(i interface{}) error {
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
	case *commandspb.InputData_RestoreSnapshotSubmission:
		underlyingCmd, ok := i.(*commandspb.RestoreSnapshot)
		if !ok {
			return errors.New("failed to unmarshal RestoreSnapshotSubmission")
		}
		*underlyingCmd = *cmd.RestoreSnapshotSubmission
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

func (t Tx) Validate() error {
	return t.err
}

func (t Tx) BlockHeight() uint64 {
	return t.inputData.BlockHeight
}
