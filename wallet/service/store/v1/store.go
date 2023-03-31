package v1

import (
	"fmt"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/service"
	"code.vegaprotocol.io/vega/wallet/service/v1"
)

type Store struct {
	pubRsaKeyFilePath  string
	privRsaKeyFilePath string
	configFilePath     string
}

func InitialiseStore(vegaPaths paths.Paths) (*Store, error) {
	pubRsaKeyFilePath, err := vegaPaths.CreateDataPathFor(paths.WalletServicePublicRSAKeyDataFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get data path for %s: %w", paths.WalletServicePublicRSAKeyDataFile, err)
	}

	privRsaKeyFilePath, err := vegaPaths.CreateDataPathFor(paths.WalletServicePrivateRSAKeyDataFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get data path for %s: %w", paths.WalletServicePrivateRSAKeyDataFile, err)
	}

	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.WalletServiceDefaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get config path for %s: %w", paths.WalletServiceDefaultConfigFile, err)
	}

	return &Store{
		pubRsaKeyFilePath:  pubRsaKeyFilePath,
		privRsaKeyFilePath: privRsaKeyFilePath,
		configFilePath:     configFilePath,
	}, nil
}

func (s *Store) RSAKeysExists() (bool, error) {
	privKeyExists, err := vgfs.FileExists(s.privRsaKeyFilePath)
	if err != nil {
		return false, err
	}
	pubKeyExists, err := vgfs.FileExists(s.pubRsaKeyFilePath)
	if err != nil {
		return false, err
	}
	return privKeyExists && pubKeyExists, nil
}

func (s *Store) SaveRSAKeys(keys *v1.RSAKeys) error {
	if err := vgfs.WriteFile(s.privRsaKeyFilePath, keys.Priv); err != nil {
		return fmt.Errorf("unable to save private key: %w", err)
	}

	if err := vgfs.WriteFile(s.pubRsaKeyFilePath, keys.Pub); err != nil {
		return fmt.Errorf("unable to save public key: %w", err)
	}

	return nil
}

func (s *Store) GetRsaKeys() (*v1.RSAKeys, error) {
	pub, err := vgfs.ReadFile(s.pubRsaKeyFilePath)
	if err != nil {
		return nil, err
	}

	priv, err := vgfs.ReadFile(s.privRsaKeyFilePath)
	if err != nil {
		return nil, err
	}

	return &v1.RSAKeys{
		Pub:  pub,
		Priv: priv,
	}, nil
}

func (s *Store) ConfigExists() (bool, error) {
	exists, err := vgfs.FileExists(s.configFilePath)
	if err != nil {
		return false, fmt.Errorf("could not verify the service configuration file existence: %w", err)
	}

	return exists, nil
}

func (s *Store) GetConfig() (*service.Config, error) {
	if exists, err := vgfs.FileExists(s.configFilePath); err != nil {
		return nil, fmt.Errorf("could not verify the service configuration file existence: %w", err)
	} else if !exists {
		return service.DefaultConfig(), nil
	}

	config := &service.Config{}
	if err := paths.ReadStructuredFile(s.configFilePath, config); err != nil {
		return nil, fmt.Errorf("could not read the service configuration file: %w", err)
	}
	return config, nil
}

func (s *Store) SaveConfig(config *service.Config) error {
	if err := paths.WriteStructuredFile(s.configFilePath, config); err != nil {
		return fmt.Errorf("could not write the service configuration file: %w", err)
	}
	return nil
}

func (s *Store) GetServiceConfigsPath() string {
	return s.configFilePath
}
