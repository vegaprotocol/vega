// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
)

type NetworkParameter struct {
	Key      string
	Value    string
	VegaTime time.Time
}

func (np *NetworkParameter) ToProto() *vega.NetworkParameter {
	pnp := vega.NetworkParameter{
		Key:   np.Key,
		Value: np.Value,
	}
	return &pnp
}

func (np NetworkParameter) Cursor() *Cursor {
	return NewCursor(np.Key)
}

func (np NetworkParameter) ToProtoEdge(_ ...any) (*v2.NetworkParameterEdge, error) {
	return &v2.NetworkParameterEdge{
		Node:   np.ToProto(),
		Cursor: np.Cursor().Encode(),
	}, nil
}

func NetworkParameterFromProto(pnp *vega.NetworkParameter) (NetworkParameter, error) {
	np := NetworkParameter{
		Key:   pnp.Key,
		Value: pnp.Value,
	}
	return np, nil
}
