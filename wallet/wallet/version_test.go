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

package wallet_test

import (
	"testing"

	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
)

func TestVersionIsSupported(t *testing.T) {
	tcs := []struct {
		name      string
		version   uint32
		supported bool
	}{
		{
			name:      "version 0",
			version:   0,
			supported: false,
		}, {
			name:      "version 1",
			version:   1,
			supported: true,
		}, {
			name:      "version 2",
			version:   2,
			supported: true,
		}, {
			name:      "version 3",
			version:   3,
			supported: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// when
			supported := wallet.IsKeyDerivationVersionSupported(tc.version)

			assert.Equal(tt, tc.supported, supported)
		})
	}
}
