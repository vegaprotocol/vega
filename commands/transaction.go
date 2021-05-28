package commands

import (
	"encoding/hex"
	"errors"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/protobuf/proto"
)

var (
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrCannotDecodeSignature = errors.New("cannot decode signature")
)

func CheckTransaction(tx *commandspb.Transaction) error {
	errs := NewErrors()
	if len(tx.InputData) == 0 {
		errs.AddForProperty("tx.input_data", ErrIsRequired)
	}
	if tx.Signature == nil {
		errs.AddForProperty("tx.signature", ErrIsRequired)
	} else {
		errs.Merge(checkSignature(tx.Signature))
	}
	if tx.From == nil {
		errs.AddForProperty("tx.from", ErrIsRequired)
	} else if len(tx.GetPubKey()) == 0 {
		errs.AddForProperty("tx.from.pub_key", ErrIsRequired)
	}

	if !errs.Empty() {
		return errs.ErrorOrNil()
	}

	errs.Merge(validateSignature(tx.InputData, tx.Signature, tx.GetPubKey()))
	if !errs.Empty() {
		return errs.ErrorOrNil()
	}

	errs.Merge(checkInputData(tx.InputData))

	return errs.ErrorOrNil()
}

func validateSignature(inputData []byte, signature *commandspb.Signature, pubKey string) Errors {
	errs := NewErrors()

	validator, err := crypto.NewSignatureAlgorithm(signature.Algo)
	if err != nil {
		return errs.FinalAddForProperty("tx.signature.algo", err)
	}

	decodedSig, err := hex.DecodeString(signature.Bytes)
	if err != nil {
		return errs.FinalAddForProperty("tx.signature.bytes", ErrCannotDecodeSignature)
	}

	decodePubKey, err := hex.DecodeString(pubKey)
	if err != nil {
		return errs.FinalAddForProperty("tx.from.pub_key", ErrCannotDecodeSignature)
	}

	ok, err := validator.Verify(decodePubKey, inputData, decodedSig)
	if err != nil {
		return errs.FinalAdd(err)
	}
	if !ok {
		return errs.FinalAddForProperty("tx.signature", ErrInvalidSignature)
	}
	return errs
}

func checkInputData(inputData []byte) Errors {
	errs := NewErrors()

	input := commandspb.InputData{}
	err := proto.Unmarshal(inputData, &input)
	if err != nil {
		return errs.FinalAdd(err)
	}

	if input.Nonce == 0 {
		errs.AddForProperty("tx.input_data.nonce", ErrMustBePositive)
	}

	if input.Command == nil {
		errs.AddForProperty("tx.input_data.command", ErrIsRequired)
	} else {
		switch cmd := input.Command.(type) {
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
		case *commandspb.InputData_ProposalSubmission:
			errs.Merge(checkProposalSubmission(cmd.ProposalSubmission))
		case *commandspb.InputData_NodeRegistration:
			errs.Merge(checkNodeRegistration(cmd.NodeRegistration))
		case *commandspb.InputData_NodeVote:
			errs.Merge(checkNodeVote(cmd.NodeVote))
		case *commandspb.InputData_NodeSignature:
			errs.Merge(checkNodeSignature(cmd.NodeSignature))
		case *commandspb.InputData_ChainEvent:
			errs.Merge(checkChainEvent(cmd.ChainEvent))
		case *commandspb.InputData_OracleDataSubmission:
			errs.Merge(checkOracleDataSubmission(cmd.OracleDataSubmission))
		default:
			errs.AddForProperty("tx.input_data.command", ErrIsNotSupported)
		}
	}

	return errs
}

func checkSignature(signature *commandspb.Signature) Errors {
	errs := NewErrors()
	if len(signature.Bytes) == 0 {
		errs.AddForProperty("tx.signature.bytes", ErrIsRequired)
	}
	if len(signature.Algo) == 0 {
		errs.AddForProperty("tx.signature.algo", ErrIsRequired)
	}
	return errs
}
