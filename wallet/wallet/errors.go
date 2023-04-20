package wallet

import (
	"errors"
	"fmt"
)

var (
	ErrAllKeysInWalletAreTainted          = errors.New("all the keys in this wallet are tainted")
	ErrInvalidRecoveryPhrase              = errors.New("the recovery phrase is not valid")
	ErrIsolatedWalletCantGenerateKeys     = errors.New("an isolated wallet can't generate keys")
	ErrIsolatedWalletDoesNotHaveMasterKey = errors.New("an isolated wallet doesn't have a master key")
	ErrPubKeyAlreadyTainted               = errors.New("the public key is already tainted")
	ErrPubKeyDoesNotExist                 = errors.New("the public key does not exist")
	ErrPubKeyIsTainted                    = errors.New("the public key is tainted")
	ErrPubKeyNotTainted                   = errors.New("the public key is not tainted")
	ErrWalletAlreadyExists                = errors.New("a wallet with the same name already exists")
	ErrWalletDoesNotExist                 = errors.New("the wallet does not exist")
	ErrWalletDoesNotHaveKeys              = errors.New("the wallet does not have keys")
	ErrWrongPassphrase                    = errors.New("wrong passphrase")
)

type UnsupportedWalletVersionError struct {
	UnsupportedVersion uint32
}

func NewUnsupportedWalletVersionError(v uint32) UnsupportedWalletVersionError {
	return UnsupportedWalletVersionError{
		UnsupportedVersion: v,
	}
}

func (e UnsupportedWalletVersionError) Error() string {
	return fmt.Sprintf("the wallet with key derivation version %d isn't supported", e.UnsupportedVersion)
}
