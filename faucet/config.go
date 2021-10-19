package faucet

import (
	"fmt"
	"os"
	"time"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config/encoding"
	vghttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger     = "faucet"
	defaultCoolDown = 1 * time.Minute
)

type Config struct {
	Level      encoding.LogLevel      `long:"level" description:"Log level"`
	RateLimit  vghttp.RateLimitConfig `group:"RateLimit" namespace:"rateLimit"`
	WalletName string                 `long:"wallet-name" description:"Name of the wallet to use to sign events"`
	Port       int                    `long:"port" description:"Listen for connections on port <port>"`
	IP         string                 `long:"ip" description:"Bind to address <ip>"`
	Node       NodeConfig             `group:"Node" namespace:"node"`
}

type NodeConfig struct {
	Port    int    `long:"port" description:"Connect to Node on port <port>"`
	IP      string `long:"ip" description:"Connect to Node on address <ip>"`
	Retries uint64 `long:"retries" description:"Connection retries before fail"`
}

func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		RateLimit: vghttp.RateLimitConfig{
			CoolDown:  encoding.Duration{Duration: defaultCoolDown},
			AllowList: []string{"10.0.0.0/8", "127.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fe80::/10"},
		},
		Node: NodeConfig{
			IP:      "127.0.0.1",
			Port:    3002,
			Retries: 5,
		},
		IP:   "0.0.0.0",
		Port: 1790,
	}
}

type ConfigLoader struct {
	configFilePath string
}

func InitialiseConfigLoader(vegaPaths paths.Paths) (*ConfigLoader, error) {
	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.FaucetDefaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get path for %s: %w", paths.FaucetDefaultConfigFile, err)
	}

	return &ConfigLoader{
		configFilePath: configFilePath,
	}, nil
}

func (l *ConfigLoader) ConfigFilePath() string {
	return l.configFilePath
}

func (l *ConfigLoader) ConfigExists() (bool, error) {
	exists, err := vgfs.FileExists(l.configFilePath)
	if err != nil {
		return false, fmt.Errorf("couldn't verify file presence: %w", err)
	}
	return exists, nil
}

func (l *ConfigLoader) GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := paths.ReadStructuredFile(l.configFilePath, cfg); err != nil {
		return nil, fmt.Errorf("couldn't read file at %s: %w", l.configFilePath, err)
	}
	return cfg, nil
}

func (l *ConfigLoader) SaveConfig(cfg *Config) error {
	if err := paths.WriteStructuredFile(l.configFilePath, cfg); err != nil {
		return fmt.Errorf("couldn't write file at %s: %w", l.configFilePath, err)
	}
	return nil
}

func (l *ConfigLoader) RemoveConfig() {
	_ = os.RemoveAll(l.configFilePath)
}
