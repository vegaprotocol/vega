package service

import (
	"fmt"
	"time"

	vgencoding "code.vegaprotocol.io/vega/libs/encoding"
	"go.uber.org/zap"
)

type Config struct {
	LogLevel vgencoding.LogLevel `json:"logLevel"`
	Server   ServerConfig        `json:"server"`
	APIV1    APIV1Config         `json:"apiV1"`
}

type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

func (c ServerConfig) String() string {
	return fmt.Sprintf("http://%v:%v", c.Host, c.Port)
}

type APIV1Config struct {
	MaximumTokenDuration vgencoding.Duration `json:"maximumTokenDuration"`
}

func DefaultConfig() *Config {
	return &Config{
		LogLevel: vgencoding.LogLevel{
			Level: zap.InfoLevel,
		},
		Server: ServerConfig{
			Port: 1789,
			Host: "127.0.0.1",
		},
		APIV1: APIV1Config{
			MaximumTokenDuration: vgencoding.Duration{
				Duration: 168 * time.Hour,
			},
		},
	}
}
