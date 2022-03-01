package sqlstore

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

type Config struct {
	Enabled                  encoding.Bool     `long:"enabled"`
	Host                     string            `long:"host"`
	Port                     int               `long:"port"`
	Username                 string            `long:"username"`
	Password                 string            `long:"password"`
	Database                 string            `long:"database"`
	WipeOnStartup            encoding.Bool     `long:"wipe-on-startup"`
	Level                    encoding.LogLevel `long:"log-level"`
	UseEmbedded              encoding.Bool     `long:"use-embedded" description:"Use an embedded version of Postgresql for the SQL data store"`
	FanOutBufferSize         int               `long:"fan-out-buffer-size" description:"buffer size used by the fan out event source"`
	SqlEventBrokerBufferSize int               `long:"sql-broker-buffer-size" description:"the per type buffer size in the sql event broker"`
	Timeout                  encoding.Duration `long:"db-timeout" description:"Duration to wait before database requests should time out (e.g. 10s, 60s etc.)"`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:                  false,
		Host:                     "localhost",
		Port:                     5432,
		Username:                 "vega",
		Password:                 "vega",
		Database:                 "vega",
		WipeOnStartup:            true,
		Level:                    encoding.LogLevel{Level: logging.InfoLevel},
		UseEmbedded:              false,
		FanOutBufferSize:         1000,
		SqlEventBrokerBufferSize: 100,
		Timeout:                  encoding.Duration{Duration: time.Second},
	}
}
