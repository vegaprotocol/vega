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
	tcs := []struct {
		name     string
		v1       bool
		expected string
	}{
		{
			name:     "v1 scheme",
			v1:       true,
			expected: "d0d9cfac8f805bd28a8c534069157d900b8c60d29580ebbee73ad5be71d1d2c1b20d5f10339b0ff570cea9f3422c1c599bd76b99c37cd19c8a3901bd75603404",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "52e2d9005416e7afe750b4fcf69d9e8e0fe2809127f87c327f10cfa5e76da55069ef723cdc07bafcfb4e44798f2d1b52cf618787cbccf19c6c8f2d1cd0530906",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			signer := testSigner{}
			pool := bridges.NewERC20AssetPool(signer, erc20AssetPool, chainID, tc.v1)
			sig, err := pool.SetBridgeAddress(
				erc20AssetAddr,
				num.NewUint(42),
			)

			assert.NoError(t, err)
			assert.NotNil(t, sig.Message)
			assert.NotNil(t, sig.Signature)
			assert.True(t, signer.Verify(sig.Message, sig.Signature))
			assert.Equal(t, tc.expected, sig.Signature.Hex())
		})
	}
}
