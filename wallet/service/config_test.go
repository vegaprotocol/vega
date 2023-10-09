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

package service_test

import (
	"testing"

	"code.vegaprotocol.io/vega/libs/encoding"
	"code.vegaprotocol.io/vega/wallet/service"
	"github.com/stretchr/testify/assert"
)

func TestServiceConfig(t *testing.T) {
	t.Run("test default config valid", testDefaultConfigValid)
	t.Run("test invalid configurations", testInvalidConfigurations)
}

func testDefaultConfigValid(t *testing.T) {
	// setup
	cfg := service.DefaultConfig()
	assert.NoError(t, cfg.Validate())
}

func testInvalidConfigurations(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr error
		cfg         service.Config
	}{
		{
			name:        "unset hostname",
			expectedErr: service.ErrServerHostUnset,
			cfg: service.Config{
				Server: service.ServerConfig{
					Port: 1789,
				},
			},
		},
		{
			name:        "unset port",
			expectedErr: service.ErrServerPortUnset,
			cfg: service.Config{
				Server: service.ServerConfig{
					Host: "localhost",
				},
			},
		},
		{
			name:        "invalid log level",
			expectedErr: service.ErrInvalidLogLevelValue,
			cfg: service.Config{
				Server: service.ServerConfig{
					Host: "localhost",
					Port: 1789,
				},
				LogLevel: encoding.LogLevel{
					Level: -100,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.cfg.Validate(), tt.expectedErr)
		})
	}
}
