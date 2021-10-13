package bridges

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Signer interface {
	Sign([]byte) ([]byte, error)
}

type SignaturePayload struct {
	Message   Bytes
	Signature Bytes
}

type Bytes []byte

func (b Bytes) Bytes() []byte {
	return b
}

func (b Bytes) Hex() string {
	return hex.EncodeToString(b)
}

func packBufAndSubmitter(
	buf []byte, submitter string,
) ([]byte, error) {
	typBytes, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}

	submitterAddr := ethcmn.HexToAddress(submitter)
	args2 := abi.Arguments([]abi.Argument{
		{
			Name: "bytes",
			Type: typBytes,
		},
		{
			Name: "address",
			Type: typAddr,
		},
	})

	return args2.Pack(buf, submitterAddr)
}

func sign(signer Signer, msg []byte) (*SignaturePayload, error) {
	// hash our message before signing it
	hash := crypto.Keccak256(msg)
	sig, err := signer.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("could not sign message with ethereum wallet: %w", err)
	}
	return &SignaturePayload{
		Message:   msg,
		Signature: sig,
	}, nil
}
