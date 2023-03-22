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

package validators_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/validators"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestTendermintKey(t *testing.T) {
	notBase64 := "170ffakjde"
	require.Error(t, validators.VerifyTendermintKey(notBase64))

	validKey := "794AFpbqJvHF711mhAK3fvSLnoXuuiig2ecrdeSJ/bk="
	require.NoError(t, validators.VerifyTendermintKey(validKey))
}

func TestSignVerifyAnnounceNode(t *testing.T) {
	nodeWallets := createTestNodeWallets(t)
	cmd := commandspb.AnnounceNode{
		Id:              nodeWallets.Vega.ID().Hex(),
		VegaPubKey:      nodeWallets.Vega.PubKey().Hex(),
		VegaPubKeyIndex: nodeWallets.Vega.Index(),
		ChainPubKey:     "794AFpbqJvHF711mhAK3fvSLnoXuuiig2ecrdeSJ/bk=",
		EthereumAddress: nodeWallets.Ethereum.PubKey().Hex(),
		FromEpoch:       1,
		InfoUrl:         "www.some.com",
		Name:            "that is not my name",
		AvatarUrl:       "www.avatar.com",
		Country:         "some country",
	}
	err := validators.SignAnnounceNode(&cmd, nodeWallets.Vega, nodeWallets.Ethereum)
	require.NoError(t, err)

	// verify that the expected signature for vega key is there
	messageToSign := cmd.Id + cmd.VegaPubKey + fmt.Sprintf("%d", cmd.VegaPubKeyIndex) + cmd.ChainPubKey + cmd.EthereumAddress + fmt.Sprintf("%d", cmd.FromEpoch) + cmd.InfoUrl + cmd.Name + cmd.AvatarUrl + cmd.Country
	sig, err := nodeWallets.Vega.Sign([]byte(messageToSign))
	sigHex := hex.EncodeToString(sig)
	require.NoError(t, err)
	require.Equal(t, sigHex, cmd.VegaSignature.Value)

	// verify that the expected signature for eth key is there
	ethSig, err := nodeWallets.Ethereum.Sign(crypto.Keccak256([]byte(messageToSign)))
	ethSigHex := hex.EncodeToString(ethSig)
	require.NoError(t, err)
	require.Equal(t, ethSigHex, cmd.EthereumSignature.Value)

	// test verification
	err = validators.VerifyAnnounceNode(&cmd)
	require.NoError(t, err)
}

func createTestNodeWallets(t *testing.T) *nodewallets.NodeWallets {
	t.Helper()
	config := nodewallets.NewDefaultConfig()
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)

	if _, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletsPass, false); err != nil {
		panic("couldn't generate Ethereum node wallet for tests")
	}

	if _, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletsPass, false); err != nil {
		panic("couldn't generate Vega node wallet for tests")
	}
	nw, err := nodewallets.GetNodeWallets(config, vegaPaths, registryPass)
	require.NoError(t, err)
	return nw
}

func TestVerifyAnnounceNode(t *testing.T) {
}
