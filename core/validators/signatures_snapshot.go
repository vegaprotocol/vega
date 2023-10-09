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
