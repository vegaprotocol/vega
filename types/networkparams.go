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

package types

import (
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
)

type NetworkParameter struct {
	Key, Value string
}

func NetworkParameterFromProto(p *proto.NetworkParameter) *NetworkParameter {
	return &NetworkParameter{
		Key:   p.Key,
		Value: p.Value,
	}
}

func (n NetworkParameter) IntoProto() *proto.NetworkParameter {
	return &proto.NetworkParameter{
		Key:   n.Key,
		Value: n.Value,
	}
}

func (n NetworkParameter) String() string {
	return fmt.Sprintf(
		"key(%s) value(%s)",
		n.Key,
		n.Value,
	)
}

func (n NetworkParameter) DeepClone() *NetworkParameter {
	return &NetworkParameter{
		Key:   n.Key,
		Value: n.Value,
	}
}
