package crypto

import (
	"errors"

	wcrypto "code.vegaprotocol.io/vegawallet/crypto"
	"github.com/ethereum/go-ethereum/common"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrAddressesDoesNotMatch = errors.New("addresses does not match")
	ErrInvalidSignature      = errors.New("invalid signature")
)

func VerifyEthereumSignature(message, signature []byte, hexAddress string) error {
	address := common.HexToAddress(hexAddress)
	hash := ecrypto.Keccak256(message)

	// get the pubkey from the signature
	pubkey, err := ecrypto.SigToPub(hash, signature)
	if err != nil {
		return err
	}

	// verify the signature
	signatureNoID := signature[:len(signature)-1]
	if !ecrypto.VerifySignature(ecrypto.CompressPubkey(pubkey), hash, signatureNoID) {
		return ErrInvalidSignature
	}

	// ensure the signer is the expected ethereum wallet
	signerAddress := ecrypto.PubkeyToAddress(*pubkey)
	if address != signerAddress {
		return ErrAddressesDoesNotMatch
	}

	return nil
}

func VerifyVegaSignature(message, signature, pubkey []byte) error {
	alg := wcrypto.NewEd25519()
	ok, err := alg.Verify(pubkey, message, signature)
	if err != nil {
		return err
	}

	if !ok {
		return ErrInvalidSignature
	}

	return nil
}
