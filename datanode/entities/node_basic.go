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
