package service

import (
	"errors"
	"fmt"
	"time"

	vgencoding "code.vegaprotocol.io/vega/libs/encoding"
	"go.uber.org/zap"
)

var (
	ErrInvalidLogLevelValue        = errors.New("the service log level is invalid")
	ErrInvalidMaximumTokenDuration = errors.New("the maximum token duration is invalid")
	ErrInvalidHealtCheckTTL        = errors.New("the health check ttl is invalid")
	ErrServerHostUnset             = errors.New("the service host is unset")
	ErrServerPortUnset             = errors.New("the service port is unset")
)

type Config struct {
	LogLevel vgencoding.LogLevel `json:"logLevel"`
	Server   ServerConfig        `json:"server"`
	APIV1    APIV1Config         `json:"apiV1"`
	APIV2    APIV2Config
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

type APIV2Config struct {
	HealthCheckTTL vgencoding.Duration `json:"healthCheckTTL"`
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
		APIV2: APIV2Config{
			HealthCheckTTL: vgencoding.Duration{
				Duration: 5 * time.Second,
			},
		},
	}
}

// Validate checks the values set in the server config file returning an error is anything is awry.
func (c *Config) Validate() error {
	if c.Server.Host == "" {
		return ErrServerHostUnset
	}

	if c.Server.Port == 0 {
		return ErrServerPortUnset
	}

	tokenExpiry := &vgencoding.Duration{}
	if err := tokenExpiry.UnmarshalText([]byte(c.APIV1.MaximumTokenDuration.String())); err != nil {
		return ErrInvalidMaximumTokenDuration
	}

	healthCheckTTL := &vgencoding.Duration{}
	if err := healthCheckTTL.UnmarshalText([]byte(c.APIV2.HealthCheckTTL.String())); err != nil {
		return ErrInvalidMaximumTokenDuration
	}

	logLevel := &vgencoding.LogLevel{}
	if err := logLevel.UnmarshalText([]byte(c.LogLevel.String())); err != nil {
		return ErrInvalidLogLevelValue
	}

	return nil
}
