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

package sqlstore

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

type Config struct {
	ConnectionConfig      ConnectionConfig      `group:"ConnectionConfig" namespace:"ConnectionConfig"`
	WipeOnStartup         encoding.Bool         `long:"wipe-on-startup"`
	Level                 encoding.LogLevel     `long:"log-level"`
	UseEmbedded           encoding.Bool         `long:"use-embedded" description:"Use an embedded version of Postgresql for the SQL data store"`
	FanOutBufferSize      int                   `long:"fan-out-buffer-size" description:"buffer size used by the fan out event source"`
	RetentionPolicies     []RetentionPolicy     `group:"RetentionPolicies" namespace:"RetentionPolicies"`
	ConnectionRetryConfig ConnectionRetryConfig `group:"ConnectionRetryConfig" namespace:"ConnectionRetryConfig"`
	LogRotationConfig     LogRotationConfig     `group:"LogRotationConfig" namespace:"LogRotationConfig"`
}

type ConnectionConfig struct {
	Host                  string            `long:"host"`
	Port                  int               `long:"port"`
	Username              string            `long:"username"`
	Password              string            `long:"password"`
	Database              string            `long:"database"`
	SocketDir             string            `long:"socket-dir" description:"location of postgres UNIX socket directory (used if host is empty string)"`
	MaxConnLifetime       encoding.Duration `long:"max-conn-lifetime"`
	MaxConnLifetimeJitter encoding.Duration `long:"max-conn-lifetime-jitter"`
}

type RetentionPolicy struct {
	HypertableOrCaggName string `string:"hypertable-or-cagg-name" description:"the name of the hyper table of continuous aggregate (cagg) to which this policy applies"`
	DataRetentionPeriod  string `string:"interval" description:"the period to retain data, e.g '3 days', '3 months', '1 year' etc"`
}

type ConnectionRetryConfig struct {
	MaxRetries      uint64        `long:"max-retries" description:"the maximum number of times to retry connecting to the database"`
	InitialInterval time.Duration `long:"initial-interval" description:"the initial interval to wait before retrying"`
	MaxInterval     time.Duration `long:"max-interval" description:"the maximum interval to wait before retrying"`
	MaxElapsedTime  time.Duration `long:"max-elapsed-time" description:"the maximum elapsed time to wait before giving up"`
}

type LogRotationConfig struct {
	MaxSize int `long:"max-size" description:"the maximum size of the log file in bytes"`
	MaxAge  int `long:"max-age" description:"the maximum number of days to keep the log file"`
}

func (conf ConnectionConfig) GetConnectionString() string {
	return conf.getConnectionStringForDatabase(conf.Database)
}

func (conf ConnectionConfig) getConnectionStringForDatabase(database string) string {
	if conf.Host == "" {
		//nolint:nosprintfhostport
		return fmt.Sprintf("postgresql://%s:%s@/%s?host=%s&port=%d",
			conf.Username,
			conf.Password,
			database,
			conf.SocketDir,
			conf.Port)
	}
	//nolint:nosprintfhostport
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		conf.Username,
		conf.Password,
		conf.Host,
		conf.Port,
		database)
}

func (conf ConnectionConfig) GetConnectionStringForPostgresDatabase() string {
	return conf.getConnectionStringForDatabase("postgres")
}

func (conf ConnectionConfig) GetPoolConfig() (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(conf.GetConnectionString())
	cfg.MaxConnLifetime = conf.MaxConnLifetime.Duration
	cfg.MaxConnLifetimeJitter = conf.MaxConnLifetimeJitter.Duration
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["application_name"] = "Vega Data Node"
	return cfg, nil
}

func NewDefaultConfig() Config {
	return Config{
		ConnectionConfig: ConnectionConfig{
			Host:                  "localhost",
			Port:                  5432,
			Username:              "vega",
			Password:              "vega",
			Database:              "vega",
			SocketDir:             "/tmp",
			MaxConnLifetime:       encoding.Duration{Duration: time.Minute * 30},
			MaxConnLifetimeJitter: encoding.Duration{Duration: time.Minute * 5},
		},
		WipeOnStartup:    true,
		Level:            encoding.LogLevel{Level: logging.InfoLevel},
		UseEmbedded:      false,
		FanOutBufferSize: 1000,
		RetentionPolicies: []RetentionPolicy{
			{HypertableOrCaggName: "balances", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "checkpoints", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "conflated_balances", DataRetentionPeriod: "1 year"},
			{HypertableOrCaggName: "delegations", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "ledger", DataRetentionPeriod: "6 months"},
			{HypertableOrCaggName: "orders", DataRetentionPeriod: "1 month"},
			{HypertableOrCaggName: "trades", DataRetentionPeriod: "1 year"},
			{HypertableOrCaggName: "trades_candle_1_minute", DataRetentionPeriod: "1 month"},
			{HypertableOrCaggName: "trades_candle_5_minutes", DataRetentionPeriod: "1 month"},
			{HypertableOrCaggName: "trades_candle_15_minutes", DataRetentionPeriod: "1 month"},
			{HypertableOrCaggName: "trades_candle_1_hour", DataRetentionPeriod: "1 year"},
			{HypertableOrCaggName: "trades_candle_6_hours", DataRetentionPeriod: "1 year"},
			{HypertableOrCaggName: "market_data", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "margin_levels", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "conflated_margin_levels", DataRetentionPeriod: "1 year"},
			{HypertableOrCaggName: "positions", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "conflated_positions", DataRetentionPeriod: "1 year"},
			{HypertableOrCaggName: "liquidity_provisions", DataRetentionPeriod: "1 day"},
		},
		ConnectionRetryConfig: ConnectionRetryConfig{
			MaxRetries:      10,
			InitialInterval: time.Second,
			MaxInterval:     time.Second * 10,
			MaxElapsedTime:  time.Minute,
		},
		LogRotationConfig: LogRotationConfig{
			MaxSize: 100,
			MaxAge:  2,
		},
	}
}
