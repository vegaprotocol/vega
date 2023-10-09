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

package bridges_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

const (
	erc20AssetPool = "0xcB84d72e61e383767C4DFEb2d8ff7f4FB89abc6e"
)

func TestAssetPoolSetBridgeAddress(t *testing.T) {
	signer := testSigner{}
	pool := bridges.NewERC20AssetPool(signer, erc20AssetPool)
	sig, err := pool.SetBridgeAddress(
		erc20AssetAddr,
		num.NewUint(42),
	)

	assert.NoError(t, err)
	assert.NotNil(t, sig.Message)
	assert.NotNil(t, sig.Signature)
	assert.True(t, signer.Verify(sig.Message, sig.Signature))
	assert.Equal(t,
		"2488c05dd36a754db037f22a1d649109573e299a3c135efdb81c6f64632b26101c0b4ce19c896d370abae8d457682b21a4a3322f48380f29932b311b6ab47707",
		sig.Signature.Hex(),
	)
}
