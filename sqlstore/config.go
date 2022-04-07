package sqlstore

import (
	"fmt"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Config struct {
	Enabled          encoding.Bool     `long:"enabled"`
	ConnectionConfig ConnectionConfig  `group:"ConnectionConfig" namespace:"ConnectionConfig"`
	WipeOnStartup    encoding.Bool     `long:"wipe-on-startup"`
	Level            encoding.LogLevel `long:"log-level"`
	UseEmbedded      encoding.Bool     `long:"use-embedded" description:"Use an embedded version of Postgresql for the SQL data store"`
	FanOutBufferSize int               `long:"fan-out-buffer-size" description:"buffer size used by the fan out event source"`
}

type ConnectionConfig struct {
	Host            string        `long:"host"`
	Port            int           `long:"port"`
	Username        string        `long:"username"`
	Password        string        `long:"password"`
	Database        string        `long:"database"`
	UseTransactions encoding.Bool `long:"use-transactions" description:"If true all changes caused by events in a single block will be committed in a single transaction"`
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
		Enabled: false,
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
	}
}
