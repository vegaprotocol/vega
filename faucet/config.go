package faucet

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"github.com/zannen/toml"
)

const (
	namedLogger     = "faucet"
	defaultWallet   = "faucet-wallet"
	configFile      = "faucet.toml"
	defaultCoolDown = 1 * time.Minute
)

type Config struct {
	Level      encoding.LogLevel
	CoolDown   encoding.Duration
	WalletPath string
	Port       int
	IP         string
	Node       NodeConfig
}

type NodeConfig struct {
	Port    int
	IP      string
	Retries uint64
}

func NewDefaultConfig(defaultDirPath string) Config {
	return Config{
		Level:      encoding.LogLevel{Level: logging.InfoLevel},
		CoolDown:   encoding.Duration{Duration: defaultCoolDown},
		WalletPath: filepath.Join(defaultDirPath, defaultWallet),
		Node: NodeConfig{
			IP:      "127.0.0.1",
			Port:    3002,
			Retries: 5,
		},
		IP:   "0.0.0.0",
		Port: 1790,
	}
}

func LoadConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(filepath.Join(path, configFile))
	if err != nil {
		return nil, err
	}
	cfg := Config{}
	if _, err := toml.Decode(string(buf), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func GenConfig(log *logging.Logger, path, passphrase string, rewrite bool) (string, error) {
	confPath := filepath.Join(path, configFile)
	confPathExists, _ := fsutil.PathExists(confPath)
	if confPathExists {
		if rewrite {
			log.Info("removing existing configuration",
				logging.String("path", confPath))
			err := os.Remove(confPath)
			if err != nil {
				return "", fmt.Errorf("unable to remove configuration: %v", err)
			}
		} else {
			// file exist, but not allowed to rewrite, return an error
			return "", fmt.Errorf("configuration already exists at path: %v", confPath)
		}
	}

	walletPath := filepath.Join(path, defaultWallet)
	confPathExists, _ = fsutil.PathExists(walletPath)
	if confPathExists {
		if rewrite {
			log.Info("removing existing configuration",
				logging.String("path", walletPath))
			err := os.Remove(walletPath)
			if err != nil {
				return "", fmt.Errorf("unable to remove configuration: %v", err)
			}
		} else {
			// file exist, but not allowed to rewrite, return an error
			return "", fmt.Errorf("configuration already exists at path: %v", walletPath)
		}
	}

	cfg := NewDefaultConfig(path)

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(cfg); err != nil {
		return "", err
	}

	// create the configuration file
	f, err := os.Create(confPath)
	if err != nil {
		return "", err
	}

	if _, err = f.WriteString(buf.String()); err != nil {
		return "", err
	}

	f.Chmod(0600)
	f.Close()

	log.Info("faucet configuration generated successfully", logging.String("path", confPath))

	// then we generate the wallet
	pubkey, err := Init(walletPath, passphrase)
	if err != nil {
		return "", err
	}

	return pubkey, nil
}
