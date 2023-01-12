package service

import (
	"fmt"

	"code.vegaprotocol.io/vega/wallet/service/v1"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/store_mock.go -package mocks code.vegaprotocol.io/vega/wallet/service Store
type Store interface {
	GetRsaKeys() (*v1.RSAKeys, error)
	RSAKeysExists() (bool, error)
	SaveRSAKeys(*v1.RSAKeys) error
}

func InitialiseService(store Store, overwrite bool) error {
	if !overwrite {
		rsaKeysExists, err := store.RSAKeysExists()
		if err != nil {
			return fmt.Errorf("couldn't verify RSA keys existence: %w", err)
		}
		if rsaKeysExists {
			return nil
		}
	}

	keys, err := v1.GenerateRSAKeys()
	if err != nil {
		return fmt.Errorf("couldn't generate RSA keys: %w", err)
	}

	if err := store.SaveRSAKeys(keys); err != nil {
		return fmt.Errorf("couldn't save RSA keys: %w", err)
	}

	return nil
}

func IsInitialised(store Store) (bool, error) {
	return store.RSAKeysExists()
}
