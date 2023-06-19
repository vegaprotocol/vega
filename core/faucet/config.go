// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package faucet

import (
	"fmt"
	"os"
	"time"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vghttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

const (
	namedLogger     = "faucet"
	defaultCoolDown = 1 * time.Minute
)

type Config struct {
	Level      encoding.LogLevel      `description:"Log level"                                long:"level"`
	RateLimit  vghttp.RateLimitConfig `group:"RateLimit"                                      namespace:"rateLimit"`
	WalletName string                 `description:"Name of the wallet to use to sign events" long:"wallet-name"`
	Port       int                    `description:"Listen for connections on port <port>"    long:"port"`
	IP         string                 `description:"Bind to address <ip>"                     long:"ip"`
	Node       NodeConfig             `group:"Node"                                           namespace:"node"`
}

type NodeConfig struct {
	Port    int    `description:"Connect to Node on port <port>"  long:"port"`
	IP      string `description:"Connect to Node on address <ip>" long:"ip"`
	Retries uint64 `description:"Connection retries before fail"  long:"retries"`
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
