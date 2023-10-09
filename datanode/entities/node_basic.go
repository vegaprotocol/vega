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

type NodeBasic struct {
	ID              NodeID
	PubKey          VegaPublicKey       `db:"vega_pub_key"`
	TmPubKey        TendermintPublicKey `db:"tendermint_pub_key"`
	EthereumAddress EthereumAddress
	InfoURL         string
	Location        string
	Status          NodeStatus
	Name            string
	AvatarURL       string
	TxHash          TxHash
	VegaTime        time.Time
}

func (n NodeBasic) ToProto() *v2.NodeBasic {
	return &v2.NodeBasic{
		Id:              n.ID.String(),
		PubKey:          n.PubKey.String(),
		TmPubKey:        n.TmPubKey.String(),
		EthereumAddress: n.EthereumAddress.String(),
		InfoUrl:         n.InfoURL,
		Location:        n.Location,
		Status:          vega.NodeStatus(n.Status),
		Name:            n.Name,
		AvatarUrl:       n.AvatarURL,
	}
}
