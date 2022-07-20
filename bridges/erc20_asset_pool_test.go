// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package bridges_test

import (
	"testing"

	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/types/num"

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
