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
	vhttp "code.vegaprotocol.io/vega/http"
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
	Level      encoding.LogLevel     `long:"level" description:"Log level"`
	RateLimit  vhttp.RateLimitConfig `group:"RateLimit" namespace:"rateLimit"`
	WalletPath string                `long:"wallet-path" description:" "`
	Port       int                   `long:"port" description:"Listen for connections on port <port>"`
	IP         string                `long:"ip" description:"Bind to address <ip>"`
	Node       NodeConfig            `group:"Node" namespace:"node"`
}

type NodeConfig struct {
	Port    int    `long:"port" description:"Connect to Node on port <port>"`
	IP      string `long:"ip" description:"Connect to Node on address <ip>"`
	Retries uint64 `long:"retries" description:"Connection retries before fail"`
}

func NewDefaultConfig(defaultDirPath string) Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		RateLimit: vhttp.RateLimitConfig{
			CoolDown:  encoding.Duration{Duration: defaultCoolDown},
			AllowList: []string{"10.0.0.0/8", "127.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fe80::/10"},
		},
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
