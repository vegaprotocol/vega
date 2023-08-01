package commands_test

import (
	"encoding/hex"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckAnnounceNode(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilAnnounceNodeFails)
	t.Run("Submitting a node registration without vega pub key fails", testAnnounceNodeWithoutVegaPubKeyFails)
	t.Run("Submitting a node registration with valid vega pub key succeeds", testAnnounceNodeWithValidVegaPubKeySucceeds)
	t.Run("Submitting a node registration with invalid encoding of vega pub key succeeds", testAnnounceNodeWithInvalidEncodingOfVegaPubKeyFails)
	t.Run("Submitting a node registration without ethereum pub key fails", testAnnounceNodeWithoutEthereumAddressFails)
	t.Run("Submitting a node registration with ethereum address succeeds", testAnnounceNodeWithEthereumAddressSucceeds)
	t.Run("Submitting a node registration without chain address fails", testAnnounceNodeWithoutChainPubKeyFails)
	t.Run("Submitting a node registration with chain pub key succeeds", testAnnounceNodeWithChainPubKeySucceeds)
	t.Run("Submitting a node registration with empty signatures fails", testAnnounceNodeWithEmptySignaturesFails)
	t.Run("Submitting a node registration with nonhex signatures fails", testAnnounceNodeWithNonhexSignaturesFails)
}

func testNilAnnounceNodeFails(t *testing.T) {
	err := checkAnnounceNode(nil)

	assert.Error(t, err)
}

func testAnnounceNodeWithoutVegaPubKeyFails(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{})
	assert.Contains(t, err.Get("announce_node.vega_pub_key"), commands.ErrIsRequired)
}

func testAnnounceNodeWithValidVegaPubKeySucceeds(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{
		VegaPubKey: hex.EncodeToString([]byte("0xDEADBEEF")),
	})
	assert.NotContains(t, err.Get("announce_node.vega_pub_key"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("announce_node.vega_pub_key"), commands.ErrShouldBeHexEncoded)
}

func testAnnounceNodeWithInvalidEncodingOfVegaPubKeyFails(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{
		VegaPubKey: "invalid-hex-encoding",
	})
	assert.Contains(t, err.Get("announce_node.vega_pub_key"), commands.ErrShouldBeAValidVegaPublicKey)
}

func testAnnounceNodeWithoutEthereumAddressFails(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{})
	assert.Contains(t, err.Get("announce_node.ethereum_address"), commands.ErrIsRequired)
}

func testAnnounceNodeWithEthereumAddressSucceeds(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{
		EthereumAddress: "0xDEADBEEF",
	})
	assert.NotContains(t, err.Get("announce_node.ethereum_address"), commands.ErrIsRequired)
}

func testAnnounceNodeWithoutChainPubKeyFails(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{})
	assert.Contains(t, err.Get("announce_node.chain_pub_key"), commands.ErrIsRequired)
}

func testAnnounceNodeWithChainPubKeySucceeds(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{
		ChainPubKey: "0xDEADBEEF",
	})
	assert.NotContains(t, err.Get("announce_node.chain_pub_key"), commands.ErrIsRequired)
}

func testAnnounceNodeWithEmptySignaturesFails(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{})
	assert.Contains(t, err.Get("announce_node.ethereum_signature"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("announce_node.vega_signature"), commands.ErrIsRequired)
}

func testAnnounceNodeWithNonhexSignaturesFails(t *testing.T) {
	err := checkAnnounceNode(&commandspb.AnnounceNode{
		VegaSignature: &commandspb.Signature{
			Value: "hello",
		},
		EthereumSignature: &commandspb.Signature{
			Value: "helloagain",
		},
	})
	assert.Contains(t, err.Get("announce_node.ethereum_signature.value"), commands.ErrShouldBeHexEncoded)
	assert.Contains(t, err.Get("announce_node.vega_signature.value"), commands.ErrShouldBeHexEncoded)
}

func checkAnnounceNode(cmd *commandspb.AnnounceNode) commands.Errors {
	err := commands.CheckAnnounceNode(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
