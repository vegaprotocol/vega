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

package validators_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/validators"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestTendermintKey(t *testing.T) {
	t.Parallel()
	notBase64 := "170ffakjde"
	require.Error(t, validators.VerifyTendermintKey(notBase64))

	validKey := "794AFpbqJvHF711mhAK3fvSLnoXuuiig2ecrdeSJ/bk="
	require.NoError(t, validators.VerifyTendermintKey(validKey))
}

func TestAnnounceNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tt := getTestTopology(t)
	cmd := createSignedAnnounceCommand(t)

	// Now announce it and check the signature verify
	require.NoError(t, tt.Topology.ProcessAnnounceNode(ctx, cmd))

	// Announce it again
	require.ErrorIs(t, tt.Topology.ProcessAnnounceNode(ctx, cmd), validators.ErrVegaNodeAlreadyRegisterForChain)
}

func createSignedAnnounceCommand(t *testing.T) *commandspb.AnnounceNode {
	t.Helper()
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

	return &cmd
}

func createTestNodeWallets(t *testing.T) *nodewallets.NodeWallets {
	t.Helper()
	config := nodewallets.NewDefaultConfig()
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)

	if _, err := nodewallets.GenerateEthereumWallet(vegaPaths, registryPass, walletsPass, "", false); err != nil {
		t.Fatal("couldn't generate Ethereum node wallet for tests")
	}

	if _, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletsPass, false); err != nil {
		t.Fatal("couldn't generate Vega node wallet for tests")
	}
	nw, err := nodewallets.GetNodeWallets(config, vegaPaths, registryPass)
	require.NoError(t, err)
	return nw
}
