package faucet

import (
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger     = "faucet"
	defaultWallet   = "faucet-wallet"
	defaultCoolDown = 5 * time.Hour
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
