package wallet

import (
	"errors"
	"fmt"
)

var (
	ErrIsolatedWalletCantGenerateKeys     = errors.New("an isolated wallet can't generate keys")
	ErrIsolatedWalletDoesNotHaveMasterKey = errors.New("an isolated wallet doesn't have a master key")
	ErrCantRotateKeyInIsolatedWallet      = errors.New("an isolated wallet can't rotate key")
	ErrInvalidRecoveryPhrase              = errors.New("the recovery phrase is not valid")
	ErrPubKeyAlreadyTainted               = errors.New("the public key is already tainted")
	ErrPubKeyIsTainted                    = errors.New("the public key is tainted")
	ErrPubKeyNotTainted                   = errors.New("the public key is not tainted")
	ErrPubKeyDoesNotExist                 = errors.New("the public key does not exist")
	ErrWalletAlreadyExists                = errors.New("a wallet with the same name already exists")
	ErrWalletDoesNotExists                = errors.New("the wallet does not exist")
	ErrWalletNotLoggedIn                  = errors.New("the wallet is not logged in")
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
	return fmt.Sprintf("wallet with version %d isn't supported", e.UnsupportedVersion)
}
