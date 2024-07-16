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

package eth_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/client/eth"

	"github.com/stretchr/testify/assert"
)

func TestStrippedError(t *testing.T) {
	t.Run("when there is a secret in URL", func(t *testing.T) {
		origErr := fmt.Errorf("Post \\\"https://arbitrum.api.onfinality.io/rpc?token=<<<<<<<LEAKED SECRET>>>>>>\": read tcp 123.222.121.111:49122->145.222.111.111:443: read: connection reset by peer")
		err := eth.NewErrorWithStrippedSecrets(origErr)
		assert.Equal(t, "Post \\\"https://arbitrum.api.onfinality.io/rpc?token=xxx\": read tcp 123.222.121.111:49122->145.222.111.111:443: read: connection reset by peer", err.Error())
	})

	t.Run("when there is no secret in URL", func(t *testing.T) {
		origErr := fmt.Errorf("Post \\\"https://arbitrum.api.onfinality.io/rpc\": read tcp 123.222.121.111:49122->145.222.111.111:443: read: connection reset by peer")
		err := eth.NewErrorWithStrippedSecrets(origErr)
		assert.Equal(t, "Post \\\"https://arbitrum.api.onfinality.io/rpc\": read tcp 123.222.121.111:49122->145.222.111.111:443: read: connection reset by peer", err.Error())
	})
}
