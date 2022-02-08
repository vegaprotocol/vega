package sqlstore

import (
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

type Config struct {
	Enabled       encoding.Bool     `long:"enabled"`
	Host          string            `long:"host"`
	Port          int               `long:"port"`
	Username      string            `long:"username"`
	Password      string            `long:"password"`
	Database      string            `long:"database"`
	WipeOnStartup encoding.Bool     `long:"wipe-on-startup"`
	Level         encoding.LogLevel `long:"log-level"`
	UseEmbedded   encoding.Bool     `long:"use-embedded" description:"Use an embedded version of Postgresql for the SQL data store"`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:       false,
		Host:          "localhost",
		Port:          5432,
		Username:      "vega",
		Password:      "vega",
		Database:      "vega",
		WipeOnStartup: true,
		Level:         encoding.LogLevel{Level: logging.InfoLevel},
		UseEmbedded:   false,
	}
}
