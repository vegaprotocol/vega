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
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func (s *ERC20Signatures) SerialisePendingSignatures() *snapshot.ToplogySignatures {
	pending := make([]*snapshot.PendingERC20MultisigControlSignature, 0, len(s.pendingSignatures))
	for e, data := range s.pendingSignatures {
		pending = append(pending,
			&snapshot.PendingERC20MultisigControlSignature{
				NodeId:          data.NodeID,
				Nonce:           data.Nonce.String(),
				EthereumAddress: e,
				Added:           data.Added,
				EpochSeq:        data.EpochSeq,
			},
		)
	}
	sort.SliceStable(pending, func(i, j int) bool {
		return pending[i].EthereumAddress < pending[j].EthereumAddress
	})

	issued := make([]*snapshot.IssuedERC20MultisigControlSignature, 0, len(s.issuedSignatures))
	for resID, data := range s.issuedSignatures {
		issued = append(issued, &snapshot.IssuedERC20MultisigControlSignature{
			ResourceId:       resID,
			EthereumAddress:  data.EthAddress,
			SubmitterAddress: data.SubmitterAddress,
		})
	}
	sort.SliceStable(issued, func(i, j int) bool {
		return issued[i].ResourceId < issued[j].ResourceId
	})

	return &snapshot.ToplogySignatures{
		PendingSignatures: pending,
		IssuedSignatures:  issued,
	}
}

func (s *ERC20Signatures) RestorePendingSignatures(sigs *snapshot.ToplogySignatures) {
	for _, data := range sigs.PendingSignatures {
		nonce, overflow := num.UintFromString(data.Nonce, 10)
		if overflow {
			s.log.Panic("Uint string not save/restored properly", logging.String("nonce", data.Nonce))
		}
		sd := &signatureData{
			Nonce:      nonce,
			NodeID:     data.NodeId,
			EthAddress: data.EthereumAddress,
			EpochSeq:   data.EpochSeq,
			Added:      data.Added,
		}
		s.pendingSignatures[data.EthereumAddress] = sd
	}

	for _, data := range sigs.IssuedSignatures {
		s.issuedSignatures[data.ResourceId] = issuedSignature{
			EthAddress:       data.EthereumAddress,
			SubmitterAddress: data.SubmitterAddress,
		}
	}
}
