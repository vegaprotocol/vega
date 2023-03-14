package entities

import (
	"time"
)

type NodeView struct {
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
