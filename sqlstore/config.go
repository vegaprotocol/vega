package sqlstore

import (
	"code.vegaprotocol.io/data-node/config/encoding"
)

type Config struct {
	Enabled       bool              `long:"enabled"`
	Host          string            `long:"host"`
	Port          int               `long:"port"`
	Username      string            `long:"username"`
	Password      string            `long:"password"`
	Database      string            `long:"database"`
	WipeOnStartup bool              `long:"wipe-on-startup"`
	Level         encoding.LogLevel `long:"log-level"`
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
	}
}
