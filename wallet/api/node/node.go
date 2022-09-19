package node

import (
	"context"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/nodes_mocks.go -package mocks code.vegaprotocol.io/vega/wallet/api/node Node,Selector

// Node is the component used to get network information and send transactions.
type Node interface {
	Host() string
	Stop() error
	SendTransaction(context.Context, *commandspb.Transaction, apipb.SubmitTransactionRequest_Type) (string, error)
	CheckTransaction(context.Context, *commandspb.Transaction) (*apipb.CheckTransactionResponse, error)
	HealthCheck(context.Context) error
	LastBlock(context.Context) (*apipb.LastBlockHeightResponse, error)
}

// Selector implementing the strategy for node selection.
type Selector interface {
	Node(ctx context.Context) (Node, error)
	Stop()
}
