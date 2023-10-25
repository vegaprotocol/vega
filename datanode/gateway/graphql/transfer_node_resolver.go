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

package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type transferNodeResolver VegaResolverRoot

func (t *transferNodeResolver) Fees(ctx context.Context, tn *v2.TransferNode) ([]*TransferFee, error) {
	fees := make([]*TransferFee, 0, len(tn.Fees))
	for _, f := range tn.Fees {
		fees = append(fees, &TransferFee{
			TransferID: f.TransferId,
			Amount:     f.Amount,
			Epoch:      int(f.Epoch),
		})
	}
	return fees, nil
}
