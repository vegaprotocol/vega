package commands

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	wcrypto "code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/protobuf/proto"
)

const ChainIDDelimiter = '\000'

func NewTransaction(pubKey string, data []byte, signature *commandspb.Signature) *commandspb.Transaction {
	return &commandspb.Transaction{
		InputData: data,
		Signature: signature,
		From: &commandspb.Transaction_PubKey{
			PubKey: pubKey,
		},
		Version: commandspb.TxVersion_TX_VERSION_V3,
	}
}

func NewInputData(height uint64) *commandspb.InputData {
	return &commandspb.InputData{
		Nonce:       crypto.NewNonce(),
		BlockHeight: height,
	}
}

func MarshalInputData(chainID string, inputData *commandspb.InputData) ([]byte, error) {
	data, err := proto.Marshal(inputData)
	if err != nil {
		return nil, err
	}

	dataWithChainID := append([]byte(fmt.Sprintf("%s%c", chainID, ChainIDDelimiter)), data...)

	return dataWithChainID, nil
}

func UnmarshalInputData(txVersion commandspb.TxVersion, rawInputData []byte, expectedChainID string) (*commandspb.InputData, error) {
	if txVersion == commandspb.TxVersion_TX_VERSION_V3 {
		prefix := fmt.Sprintf("%s%c", expectedChainID, ChainIDDelimiter)
		if !bytes.HasPrefix(rawInputData, []byte(prefix)) {
			idx := bytes.IndexByte(rawInputData, ChainIDDelimiter)
			if idx == -1 {
				return nil, fmt.Errorf("the transaction is not bundled with a chain ID, whereas it was expecting to be %q", expectedChainID)
			}
			return nil, fmt.Errorf("the transaction as been bundled for the network %q, but the current connection is on the network %q", string(rawInputData[:idx]), expectedChainID)
		}
		rawInputData = rawInputData[len(prefix):]
	}

	inputData := &commandspb.InputData{}
	if err := proto.Unmarshal(rawInputData, inputData); err != nil {
		return nil, fmt.Errorf("couldn't unmarshall input data: %w", err)
	}
	return inputData, nil
}

func NewSignature(sig []byte, algo string, version uint32) *commandspb.Signature {
	return &commandspb.Signature{
		Value:   hex.EncodeToString(sig),
		Algo:    algo,
		Version: version,
	}
}

func CheckTransaction(tx *commandspb.Transaction, chainID string) (*commandspb.InputData, error) {
	errs := NewErrors()

	if tx == nil {
		return nil, errs.FinalAddForProperty("tx", ErrIsRequired)
	}

	if tx.Version == commandspb.TxVersion_TX_VERSION_UNSPECIFIED {
		return nil, errs.FinalAddForProperty("tx.version", ErrIsRequired)
	}
	if tx.Version != commandspb.TxVersion_TX_VERSION_V2 && tx.Version != commandspb.TxVersion_TX_VERSION_V3 {
		return nil, errs.FinalAddForProperty("tx.version", ErrIsNotSupported)
	}

	if tx.From == nil {
		errs.AddForProperty("tx.from", ErrIsRequired)
	} else if len(tx.GetPubKey()) == 0 {
		errs.AddForProperty("tx.from.pub_key", ErrIsRequired)
	} else if !IsVegaPubkey(tx.GetPubKey()) {
		errs.AddForProperty("tx.from.pub_key", ErrShouldBeAValidVegaPubkey)
	}

	// We need the above check to pass, so we verify it's all good.
	if !errs.Empty() {
		return nil, errs.ErrorOrNil()
	}

	inputData, inputErrs := checkInputData(tx.Version, tx.InputData, chainID)
	if !inputErrs.Empty() {
		errs.Merge(inputErrs)
		return nil, errs.ErrorOrNil()
	}

	errs.Merge(checkSignature(tx.Signature, tx.GetPubKey(), tx.InputData))
	if !errs.Empty() {
		return nil, errs.ErrorOrNil()
	}

	return inputData, nil
}

func checkSignature(signature *commandspb.Signature, pubKey string, rawInputData []byte) Errors {
	errs := NewErrors()

	if signature == nil {
		return errs.FinalAddForProperty("tx.signature", ErrIsRequired)
	}

	if len(signature.Value) == 0 {
		errs.AddForProperty("tx.signature.value", ErrIsRequired)
	}
	decodedSig, err := hex.DecodeString(signature.Value)
	if err != nil {
		errs.AddForProperty("tx.signature.value", ErrShouldBeHexEncoded)
	}

	if len(signature.Algo) == 0 {
		errs.AddForProperty("tx.signature.algo", ErrIsRequired)
	}
	algo, err := wcrypto.NewSignatureAlgorithm(signature.Algo, signature.Version)
	if err != nil {
		errs.AddForProperty("tx.signature.algo", ErrUnsupportedAlgorithm)
		errs.AddForProperty("tx.signature.version", ErrUnsupportedAlgorithm)
	}

	// We need the above check to pass, so we verify it's all good.
	if !errs.Empty() {
		return errs
	}

	decodedPubKey := []byte(pubKey)
	if IsVegaPubkey(pubKey) {
		// We can ignore the error has it should have been checked earlier.
		decodedPubKey, _ = hex.DecodeString(pubKey)
	}

	isValid, err := algo.Verify(decodedPubKey, rawInputData, decodedSig)
	if err != nil {
		// This shouldn't happen. If it does, we need to add better checks up-hill.
		return errs.FinalAddForProperty("tx.signature.value", ErrSignatureNotVerifiable)
	}

	if !isValid {
		errs.AddForProperty("tx.signature.value", ErrInvalidSignature)
	}

	return nil
}

func checkInputData(version commandspb.TxVersion, rawInputData []byte, expectedChainID string) (*commandspb.InputData, Errors) {
	errs := NewErrors()

	if len(rawInputData) == 0 {
		return nil, errs.FinalAddForProperty("tx.input_data", ErrIsRequired)
	}

	inputData, err := UnmarshalInputData(version, rawInputData, expectedChainID)
	if err != nil {
		return nil, errs.FinalAddForProperty("tx.input_data", err)
	}

	if inputData.Nonce == 0 {
		errs.AddForProperty("tx.input_data.nonce", ErrMustBePositive)
	}

	if inputData.Command == nil {
		errs.AddForProperty("tx.input_data.command", ErrIsRequired)
	} else {
		switch cmd := inputData.Command.(type) {
		case *commandspb.InputData_OrderSubmission:
			errs.Merge(checkOrderSubmission(cmd.OrderSubmission))
		case *commandspb.InputData_OrderCancellation:
			break // No verification to be made
		case *commandspb.InputData_OrderAmendment:
			errs.Merge(checkOrderAmendment(cmd.OrderAmendment))
		case *commandspb.InputData_VoteSubmission:
			errs.Merge(checkVoteSubmission(cmd.VoteSubmission))
		case *commandspb.InputData_WithdrawSubmission:
			errs.Merge(checkWithdrawSubmission(cmd.WithdrawSubmission))
		case *commandspb.InputData_LiquidityProvisionSubmission:
			errs.Merge(checkLiquidityProvisionSubmission(cmd.LiquidityProvisionSubmission))
		case *commandspb.InputData_LiquidityProvisionCancellation:
			errs.Merge(checkLiquidityProvisionCancellation(cmd.LiquidityProvisionCancellation))
		case *commandspb.InputData_LiquidityProvisionAmendment:
			errs.Merge(checkLiquidityProvisionAmendment(cmd.LiquidityProvisionAmendment))
		case *commandspb.InputData_ProposalSubmission:
			errs.Merge(checkProposalSubmission(cmd.ProposalSubmission))
		case *commandspb.InputData_AnnounceNode:
			errs.Merge(checkAnnounceNode(cmd.AnnounceNode))
		case *commandspb.InputData_NodeVote:
			errs.Merge(checkNodeVote(cmd.NodeVote))
		case *commandspb.InputData_NodeSignature:
			errs.Merge(checkNodeSignature(cmd.NodeSignature))
		case *commandspb.InputData_ChainEvent:
			errs.Merge(checkChainEvent(cmd.ChainEvent))
		case *commandspb.InputData_OracleDataSubmission:
			errs.Merge(checkOracleDataSubmission(cmd.OracleDataSubmission))
		case *commandspb.InputData_DelegateSubmission:
			errs.Merge(checkDelegateSubmission(cmd.DelegateSubmission))
		case *commandspb.InputData_UndelegateSubmission:
			errs.Merge(checkUndelegateSubmission(cmd.UndelegateSubmission))
		case *commandspb.InputData_KeyRotateSubmission:
			errs.Merge(checkKeyRotateSubmission(cmd.KeyRotateSubmission))
		case *commandspb.InputData_StateVariableProposal:
			errs.Merge(checkStateVariableProposal(cmd.StateVariableProposal))
		case *commandspb.InputData_TransferInstruction:
			errs.Merge(checkTransfeInstructionr(cmd.TransferInstruction))
		case *commandspb.InputData_CancelTransferInstruction:
			errs.Merge(checkCancelTransferInstruction(cmd.CancelTransferInstruction))
		case *commandspb.InputData_ValidatorHeartbeat:
			errs.Merge(checkValidatorHeartbeat(cmd.ValidatorHeartbeat))
		case *commandspb.InputData_EthereumKeyRotateSubmission:
			errs.Merge(checkEthereumKeyRotateSubmission(cmd.EthereumKeyRotateSubmission))
		case *commandspb.InputData_ProtocolUpgradeProposal:
			errs.Merge(checkProtocolUpgradeProposal(cmd.ProtocolUpgradeProposal))
		case *commandspb.InputData_IssueSignatures:
			errs.Merge(checkIssueSignatures(cmd.IssueSignatures))
		case *commandspb.InputData_BatchMarketInstructions:
			errs.Merge(checkBatchMarketInstructions(cmd.BatchMarketInstructions))
		default:
			errs.AddForProperty("tx.input_data.command", ErrIsNotSupported)
		}
	}

	return inputData, errs
}
