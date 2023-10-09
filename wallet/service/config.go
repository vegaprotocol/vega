// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"errors"
	"fmt"
	"time"

	vgencoding "code.vegaprotocol.io/vega/libs/encoding"
	"go.uber.org/zap"
)

var (
	ErrInvalidLogLevelValue              = errors.New("the service log level is invalid")
	ErrInvalidMaximumTokenDuration       = errors.New("the maximum token duration is invalid")
	ErrInvalidMaximumNodeRequestDuration = errors.New("the maximum request duration is invalid")
	ErrServerHostUnset                   = errors.New("the service host is unset")
	ErrServerPortUnset                   = errors.New("the service port is unset")
)

type Config struct {
	LogLevel vgencoding.LogLevel `json:"logLevel"`
	Server   ServerConfig        `json:"server"`
	APIV1    APIV1Config         `json:"apiV1"`
	APIV2    APIV2Config         `json:"apiV2"`
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
	Nodes Nodes `json:"nodes"`
}

type Nodes struct {
	MaximumRetryPerRequest uint64              `json:"maximumRetryPerRequest"`
	MaximumRequestDuration vgencoding.Duration `json:"maximumRequestDuration"`
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
			Nodes: Nodes{
				MaximumRetryPerRequest: 5,
				MaximumRequestDuration: vgencoding.Duration{
					Duration: 5 * time.Second,
				},
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

	requestExpiry := &vgencoding.Duration{}
	if err := requestExpiry.UnmarshalText([]byte(c.APIV2.Nodes.MaximumRequestDuration.String())); err != nil {
		return ErrInvalidMaximumNodeRequestDuration
	}

	logLevel := &vgencoding.LogLevel{}
	if err := logLevel.UnmarshalText([]byte(c.LogLevel.String())); err != nil {
		return ErrInvalidLogLevelValue
	}

	return nil
}
