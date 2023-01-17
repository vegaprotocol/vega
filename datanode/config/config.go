// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//lint:file-ignore SA5008 duplicated struct tags are ok for config

package config

import (
	"errors"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/datanode/admin"

	"code.vegaprotocol.io/vega/datanode/networkhistory"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

// Config ties together all other application configuration types.
type Config struct {
	Admin     admin.Config     `group:"Admin" namespace:"admin"`
	API       api.Config       `group:"API" namespace:"api"`
	CandlesV2 candlesv2.Config `group:"CandlesV2" namespace:"candlesv2"`
	Logging   logging.Config   `group:"Logging" namespace:"logging"`
	SQLStore  sqlstore.Config  `group:"Sqlstore" namespace:"sqlstore"`
	Gateway   gateway.Config   `group:"Gateway" namespace:"gateway"`
	Metrics   metrics.Config   `group:"Metrics" namespace:"metrics"`
	Broker    broker.Config    `group:"Broker" namespace:"broker"`
	Service   service.Config   `group:"Service" namespace:"service"`

	Pprof          pprof.Config  `group:"Pprof" namespace:"pprof"`
	GatewayEnabled encoding.Bool `long:"gateway-enabled" choice:"true" choice:"false" description:" "`

	NetworkHistory                   networkhistory.Config `group:"NetworkHistory" namespace:"networkhistory"`
	AutoInitialiseFromNetworkHistory encoding.Bool         `long:"auto-initialise" choice:"true" choice:"false" description:"if true the node will attempt to load the latest history segment(s) from network history if the node is empty"`

	ChainID          string `long:"chainID"`
	MaxMemoryPercent uint8  `long:"max-memory-percent" description:"The maximum amount of memory reserved for the data node (default: 33%)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig() Config {
	return Config{
		MaxMemoryPercent:                 33,
		Admin:                            admin.NewDefaultConfig(),
		API:                              api.NewDefaultConfig(),
		CandlesV2:                        candlesv2.NewDefaultConfig(),
		SQLStore:                         sqlstore.NewDefaultConfig(),
		Pprof:                            pprof.NewDefaultConfig(),
		Logging:                          logging.NewDefaultConfig(),
		Gateway:                          gateway.NewDefaultConfig(),
		Metrics:                          metrics.NewDefaultConfig(),
		Broker:                           broker.NewDefaultConfig(),
		Service:                          service.NewDefaultConfig(),
		GatewayEnabled:                   true,
		NetworkHistory:                   networkhistory.NewDefaultConfig(),
		AutoInitialiseFromNetworkHistory: false,
	}
}

func (c Config) GetMaxMemoryFactor() (float64, error) {
	if c.MaxMemoryPercent <= 0 || c.MaxMemoryPercent >= 100 {
		return 0, errors.New("MaxMemoryPercent is out of range, expect > 0 and < 100")
	}

	return float64(c.MaxMemoryPercent) / 100., nil
}

type Loader struct {
	configFilePath string
}

func InitialiseLoader(vegaPaths paths.Paths) (*Loader, error) {
	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get path for %s: %w", paths.NodeDefaultConfigFile, err)
	}

	return &Loader{
		configFilePath: configFilePath,
	}, nil
}

func (l *Loader) ConfigFilePath() string {
	return l.configFilePath
}

func (l *Loader) ConfigExists() (bool, error) {
	exists, err := vgfs.FileExists(l.configFilePath)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (l *Loader) Save(cfg *Config) error {
	if err := paths.WriteStructuredFile(l.configFilePath, cfg); err != nil {
		return fmt.Errorf("couldn't write configuration file at %s: %w", l.configFilePath, err)
	}
	return nil
}

func (l *Loader) Get() (*Config, error) {
	cfg := NewDefaultConfig()
	if err := paths.ReadStructuredFile(l.configFilePath, &cfg); err != nil {
		return nil, fmt.Errorf("couldn't read configuration file at %s: %w", l.configFilePath, err)
	}
	return &cfg, nil
}

func (l *Loader) Remove() {
	_ = os.RemoveAll(l.configFilePath)
}
