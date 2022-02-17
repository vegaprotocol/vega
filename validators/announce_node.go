package validators

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"github.com/ethereum/go-ethereum/crypto"
)

var ErrMissingRequiredAnnounceNodeFields = errors.New("missing required announce node fields")

func (t *Topology) ProcessAnnounceNode(
	ctx context.Context, an *commandspb.AnnounceNode) error {
	if err := VerifyAnnounceNode(an); err != nil {
		return err
	}

	t.AddNewNode(ctx, an)
	return nil
}

type Signer interface {
	Sign([]byte) ([]byte, error)
	Algo() string
}

type Verifier interface {
	Verify([]byte, []byte) error
}

// SignAnnounceNode adds the signature for the ethereum and
// Vega address / pubkeys.
func VerifyAnnounceNode(an *commandspb.AnnounceNode) error {
	buf, err := makeAnnounceNodeSignableMessage(an)
	if err != nil {
		return err
	}

	vegas, err := hex.DecodeString(an.GetVegaSignature().Value)
	if err != nil {
		return err
	}
	vegaPubKey, err := hex.DecodeString(an.GetVegaPubKey())
	if err != nil {
		return err
	}
	if err := vgcrypto.VerifyVegaSignature(buf, vegas, vegaPubKey); err != nil {
		return err
	}

	eths, err := hex.DecodeString(an.GetEthereumSignature().Value)
	if err != nil {
		return err
	}

	if err := vgcrypto.VerifyEthereumSignature(buf, eths, an.EthereumAddress); err != nil {
		return err
	}

	return nil
}

// SignAnnounceNode adds the signature for the ethereum and
// Vega address / pubkeys.
func SignAnnounceNode(
	an *commandspb.AnnounceNode,
	vegaSigner Signer,
	ethSigner Signer,
) error {
	buf, err := makeAnnounceNodeSignableMessage(an)
	if err != nil {
		return err
	}

	vegaSignature, err := vegaSigner.Sign(buf)
	if err != nil {
		return err
	}

	ethereumSignature, err := ethSigner.Sign(crypto.Keccak256(buf))
	if err != nil {
		return err
	}

	an.EthereumSignature = &commandspb.Signature{
		Value: hex.EncodeToString(ethereumSignature),
		Algo:  ethSigner.Algo(),
	}

	an.VegaSignature = &commandspb.Signature{
		Value: hex.EncodeToString(vegaSignature),
		Algo:  vegaSigner.Algo(),
	}

	return nil
}

func makeAnnounceNodeSignableMessage(an *commandspb.AnnounceNode) ([]byte, error) {
	if len(an.Id) <= 0 || len(an.VegaPubKey) <= 0 || an.VegaPubKeyIndex == 0 || len(an.ChainPubKey) <= 0 || len(an.EthereumAddress) <= 0 || an.FromEpoch == 0 || len(an.InfoUrl) <= 0 || len(an.Name) <= 0 || len(an.AvatarUrl) <= 0 || len(an.Country) <= 0 {
		return nil, ErrMissingRequiredAnnounceNodeFields
	}

	msg := an.Id + an.VegaPubKey + fmt.Sprintf("%d", an.VegaPubKeyIndex) + an.ChainPubKey + an.EthereumAddress + fmt.Sprintf("%d", an.FromEpoch) + an.InfoUrl + an.Name + an.AvatarUrl + an.Country

	return []byte(msg), nil
}
