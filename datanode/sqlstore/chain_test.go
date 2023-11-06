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

// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"github.com/stretchr/testify/assert"
)

func TestChain(t *testing.T) {
	ctx := tempTransaction(t)

	cs := sqlstore.NewChain(connectionSource)

	chain1 := entities.Chain{ID: "my-test-chain"}
	chain2 := entities.Chain{ID: "my-other-chain"}

	t.Run("fetching unset chain fails", func(t *testing.T) {
		_, err := cs.Get(ctx)
		assert.ErrorIs(t, err, entities.ErrNotFound)
	})

	t.Run("setting chain", func(t *testing.T) {
		err := cs.Set(ctx, chain1)
		assert.NoError(t, err)
	})

	t.Run("fetching set chain", func(t *testing.T) {
		fetched, err := cs.Get(ctx)
		assert.NoError(t, err)
		assert.Equal(t, fetched, chain1)
	})

	t.Run("setting chain a second time should fail", func(t *testing.T) {
		err := cs.Set(ctx, chain2)
		assert.ErrorIs(t, err, entities.ErrChainAlreadySet)
	})
}
