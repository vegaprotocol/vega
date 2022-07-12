// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"fmt"
	"os"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/service"

	"code.vegaprotocol.io/data-node/api"
	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/gateway"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/pprof"
	"code.vegaprotocol.io/data-node/sqlstore"
	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
)

// Config ties together all other application configuration types.
type Config struct {
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
	UlimitNOFile   uint64        `long:"ulimit-no-files" description:"Set the max number of open files (see: ulimit -n)" tomlcp:"Set the max number of open files (see: ulimit -n)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig() Config {
	return Config{
		API:            api.NewDefaultConfig(),
		CandlesV2:      candlesv2.NewDefaultConfig(),
		SQLStore:       sqlstore.NewDefaultConfig(),
		Pprof:          pprof.NewDefaultConfig(),
		Logging:        logging.NewDefaultConfig(),
		Gateway:        gateway.NewDefaultConfig(),
		Metrics:        metrics.NewDefaultConfig(),
		Broker:         broker.NewDefaultConfig(),
		Service:        service.NewDefaultConfig(),
		GatewayEnabled: true,
		UlimitNOFile:   8192,
	}
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
