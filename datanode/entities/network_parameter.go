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

package entities

import (
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type NetworkParameter struct {
	Key      string
	Value    string
	TxHash   TxHash
	VegaTime time.Time
}

func (np NetworkParameter) ToProto() *vega.NetworkParameter {
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

func NetworkParameterFromProto(pnp *vega.NetworkParameter, txHash TxHash) (NetworkParameter, error) {
	np := NetworkParameter{
		Key:    pnp.Key,
		Value:  pnp.Value,
		TxHash: txHash,
	}
	return np, nil
}

type NetworkParameterCursor struct {
	Key string
}

func (c *NetworkParameterCursor) Parse(cursorString string) error {
	c.Key = cursorString
	return nil
}
