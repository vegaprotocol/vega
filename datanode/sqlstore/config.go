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

package sqlstore

import (
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Config struct {
	ConnectionConfig  ConnectionConfig  `group:"ConnectionConfig" namespace:"ConnectionConfig"`
	WipeOnStartup     encoding.Bool     `long:"wipe-on-startup"`
	Level             encoding.LogLevel `long:"log-level"`
	UseEmbedded       encoding.Bool     `long:"use-embedded" description:"Use an embedded version of Postgresql for the SQL data store"`
	FanOutBufferSize  int               `long:"fan-out-buffer-size" description:"buffer size used by the fan out event source"`
	RetentionPolicies []RetentionPolicy `group:"RetentionPolicies" namespace:"RetentionPolicies"`
}

type ConnectionConfig struct {
	Host            string        `long:"host"`
	Port            int           `long:"port"`
	Username        string        `long:"username"`
	Password        string        `long:"password"`
	Database        string        `long:"database"`
	UseTransactions encoding.Bool `long:"use-transactions" description:"If true all changes caused by events in a single block will be committed in a single transaction"`
}

type RetentionPolicy struct {
	HypertableOrCaggName string `string:"hypertable-or-cagg-name" description:"the name of the hyper table of continuous aggregate (cagg) to which this policy applies"`
	DataRetentionPeriod  string `string:"interval" description:"the period to retain data, e.g '3 days', '3 months', '1 year' etc"`
}

func (conf ConnectionConfig) GetConnectionString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		conf.Username,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Database)
}

func (conf ConnectionConfig) GetPoolConfig() (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(conf.GetConnectionString())
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["application_name"] = "Vega Data Node"
	return cfg, nil
}

func NewDefaultConfig() Config {
	return Config{
		ConnectionConfig: ConnectionConfig{
			Host:            "localhost",
			Port:            5432,
			Username:        "vega",
			Password:        "vega",
			Database:        "vega",
			UseTransactions: true,
		},
		WipeOnStartup:    true,
		Level:            encoding.LogLevel{Level: logging.InfoLevel},
		UseEmbedded:      false,
		FanOutBufferSize: 1000,
		RetentionPolicies: []RetentionPolicy{
			{HypertableOrCaggName: "balances", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "conflated_balances", DataRetentionPeriod: "1 year"},
			{HypertableOrCaggName: "ledger", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "orders_history", DataRetentionPeriod: "7 days"},
			{HypertableOrCaggName: "trades", DataRetentionPeriod: "7 days"},
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
			{HypertableOrCaggName: "liquidity_provisions", DataRetentionPeriod: "1 year"},
		},
	}
}
