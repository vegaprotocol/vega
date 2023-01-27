package service

import (
	"errors"
	"fmt"

	vgencoding "code.vegaprotocol.io/vega/libs/encoding"
	"code.vegaprotocol.io/vega/wallet/service/v1"
)

var (
	ErrInvalidLogLevelValue        = errors.New("The service log level is invalid")
	ErrInvalidMaximumTokenDuration = errors.New("the maximum token duration is invalid")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/store_mock.go -package mocks code.vegaprotocol.io/vega/wallet/service Store
type Store interface {
	GetRsaKeys() (*v1.RSAKeys, error)
	RSAKeysExists() (bool, error)
	SaveRSAKeys(*v1.RSAKeys) error
	ConfigExists() (bool, error)
	SaveConfig(*Config) error
	GetConfig() (*Config, error)
}

func InitialiseService(store Store, overwrite bool) error {
	rsaKeysExists, err := store.RSAKeysExists()
	if err != nil {
		return fmt.Errorf("could not verify the RSA keys existence: %w", err)
	}
	if !rsaKeysExists || overwrite {
		keys, err := v1.GenerateRSAKeys()
		if err != nil {
			return fmt.Errorf("could not generate the RSA keys: %w", err)
		}

		if err := store.SaveRSAKeys(keys); err != nil {
			return fmt.Errorf("could not save the RSA keys: %w", err)
		}
	}

	configExists, err := store.ConfigExists()
	if err != nil {
		return fmt.Errorf("could not verify the service configuration existence: %w", err)
	}
	if !configExists || overwrite {
		if err := store.SaveConfig(DefaultConfig()); err != nil {
			return fmt.Errorf("could not save the default service configuration: %w", err)
		}
	}

	return nil
}

func IsInitialised(store Store) (bool, error) {
	rsaExists, err := store.RSAKeysExists()
	if err != nil {
		return false, fmt.Errorf("could not verify the RSA keys existence: %w", err)
	}

	configExist, err := store.ConfigExists()
	if err != nil {
		return false, fmt.Errorf("could not verify the service configuration existence: %w", err)
	}

	return rsaExists && configExist, nil
}

func UpdateConfig(store Store, cfg *Config) error {
	if err := checkConfig(cfg); err != nil {
		return fmt.Errorf("the service configuration is invalid: %w", err)
	}

	if err := store.SaveConfig(cfg); err != nil {
		return fmt.Errorf("could not save the service configuration: %w", err)
	}

	return nil
}

func checkConfig(cfg *Config) error {
	tokenExpiry := &vgencoding.Duration{}
	if err := tokenExpiry.UnmarshalText([]byte(cfg.APIV1.MaximumTokenDuration.String())); err != nil {
		return ErrInvalidMaximumTokenDuration
	}

	logLevel := &vgencoding.LogLevel{}
	if err := logLevel.UnmarshalText([]byte(cfg.LogLevel.String())); err != nil {
		return ErrInvalidLogLevelValue
	}

	return nil
}
