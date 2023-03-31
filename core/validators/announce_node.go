// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package validators

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/nodewallets/eth/clef"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	signatures "code.vegaprotocol.io/vega/libs/crypto/signature"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/ethereum/go-ethereum/crypto"
)

var ErrMissingRequiredAnnounceNodeFields = errors.New("missing required announce node fields")

func (t *Topology) ProcessAnnounceNode(
	ctx context.Context, an *commandspb.AnnounceNode,
) error {
	if err := VerifyAnnounceNode(an); err != nil {
		return err
	}

	if err := t.AddNewNode(ctx, an, ValidatorStatusPending); err != nil {
		return err
	}

	// if it is use that has annouce, we can now set our flag to be a validator. How exciting.
	if an.Id == t.SelfNodeID() {
		t.SetIsValidator()
	}
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
	// just ensure the node address is checksumed
	an.EthereumAddress = vgcrypto.EthereumChecksumAddress(an.EthereumAddress)

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
	if err := signatures.VerifyVegaSignature(buf, vegas, vegaPubKey); err != nil {
		return err
	}

	eths, err := hex.DecodeString(an.GetEthereumSignature().Value)
	if err != nil {
		return err
	}

	if err := signatures.VerifyEthereumSignature(buf, eths, an.EthereumAddress); err != nil {
		return err
	}

	if err := VerifyTendermintKey(an.ChainPubKey); err != nil {
		return err
	}

	return nil
}

func VerifyTendermintKey(tmKey string) error {
	_, err := base64.StdEncoding.DecodeString(tmKey)
	return err
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

	if ethSigner.Algo() != clef.ClefAlgoType {
		buf = crypto.Keccak256(buf)
	}
	ethereumSignature, err := ethSigner.Sign(buf)
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
