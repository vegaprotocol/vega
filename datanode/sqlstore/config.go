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

type RetentionPeriod string

const (
	RetentionPeriodStandard RetentionPeriod = "standard"
	RetentionPeriodArchive  RetentionPeriod = "forever"
	RetentionPeriodLite     RetentionPeriod = "1 day"
)

type Config struct {
	ConnectionConfig                                   ConnectionConfig      `group:"ConnectionConfig"                                                                          namespace:"ConnectionConfig"`
	WipeOnStartup                                      encoding.Bool         `description:"deprecated, use data-node unsafe_reset_all command instead"                          long:"wipe-on-startup"`
	Level                                              encoding.LogLevel     `long:"log-level"`
	UseEmbedded                                        encoding.Bool         `description:"Use an embedded version of Postgresql for the SQL data store"                        long:"use-embedded"`
	FanOutBufferSize                                   int                   `description:"buffer size used by the fan out event source"                                        long:"fan-out-buffer-size"`
	RetentionPolicies                                  []RetentionPolicy     `group:"RetentionPolicies"                                                                         namespace:"RetentionPolicies"`
	ConnectionRetryConfig                              ConnectionRetryConfig `group:"ConnectionRetryConfig"                                                                     namespace:"ConnectionRetryConfig"`
	LogRotationConfig                                  LogRotationConfig     `group:"LogRotationConfig"                                                                         namespace:"LogRotationConfig"`
	DisableMinRetentionPolicyCheckForUseInSysTestsOnly encoding.Bool         `description:"Disables the minimum retention policy interval check - only for use in system tests" long:"disable-min-retention-policy-use-in-sys-test-only"`
	RetentionPeriod                                    RetentionPeriod       `description:"Set the retention level for the database. standard, archive, or lite"                long:"retention-period"`
	VerboseMigration                                   encoding.Bool         `description:"Enable verbose logging of SQL migrations"                                            long:"verbose-migration"`
	ChunkIntervals                                     []ChunkInterval       `group:"ChunkIntervals"                                                                            namespace:"ChunkIntervals"`
}

type ConnectionConfig struct {
	Host                  string            `long:"host"`
	Port                  int               `long:"port"`
	Username              string            `long:"username"`
	Password              string            `long:"password"`
	Database              string            `long:"database"`
	SocketDir             string            `description:"location of postgres UNIX socket directory (used if host is empty string)" long:"socket-dir"`
	MaxConnLifetime       encoding.Duration `long:"max-conn-lifetime"`
	MaxConnLifetimeJitter encoding.Duration `long:"max-conn-lifetime-jitter"`
	MaxConnPoolSize       int               `long:"max-conn-pool-size"`
	MinConnPoolSize       int32             `long:"min-conn-pool-size"`
	RuntimeParams         map[string]string `long:"runtime-params"`
}

type HypertableOverride interface {
	RetentionPolicy | ChunkInterval
	EntityName() string
}

type RetentionPolicy struct {
	HypertableOrCaggName string `description:"the name of the hyper table of continuous aggregate (cagg) to which this policy applies"                          string:"hypertable-or-cagg-name"`
	DataRetentionPeriod  string `description:"the period to retain data, e.g '3 days', '3 months', '1 year' etc. To retain data indefinitely specify 'forever'" string:"interval"`
}

func (p RetentionPolicy) EntityName() string {
	return p.HypertableOrCaggName
}

type ChunkInterval struct {
	HypertableOrCaggName string `description:"the name of the hyper table of continuous aggregate (cagg) to which this policy applies" string:"hypertable-or-cagg-name"`
	ChunkInterval        string `description:"the interval at which to create new chunks, e.g '1 day', '1 month', '1 year' etc."       string:"chunk-interval"`
}

func (p ChunkInterval) EntityName() string {
	return p.HypertableOrCaggName
}

type ConnectionRetryConfig struct {
	MaxRetries      uint64        `description:"the maximum number of times to retry connecting to the database" long:"max-retries"`
	InitialInterval time.Duration `description:"the initial interval to wait before retrying"                    long:"initial-interval"`
	MaxInterval     time.Duration `description:"the maximum interval to wait before retrying"                    long:"max-interval"`
	MaxElapsedTime  time.Duration `description:"the maximum elapsed time to wait before giving up"               long:"max-elapsed-time"`
}

type LogRotationConfig struct {
	MaxSize int `description:"the maximum size of the log file in bytes"       long:"max-size"`
	MaxAge  int `description:"the maximum number of days to keep the log file" long:"max-age"`
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
	if err != nil {
		return nil, err
	}
	cfg.MaxConnLifetime = conf.MaxConnLifetime.Duration
	cfg.MaxConnLifetimeJitter = conf.MaxConnLifetimeJitter.Duration

	cfg.ConnConfig.RuntimeParams["application_name"] = "Vega Data Node"
	for paramKey, paramValue := range conf.RuntimeParams {
		cfg.ConnConfig.RuntimeParams[paramKey] = paramValue
	}
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
			RuntimeParams:         map[string]string{},
		},
		Level:            encoding.LogLevel{Level: logging.InfoLevel},
		UseEmbedded:      false,
		FanOutBufferSize: 1000,
		DisableMinRetentionPolicyCheckForUseInSysTestsOnly: false,
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
		RetentionPeriod:  RetentionPeriodStandard,
		VerboseMigration: false,
	}
}
