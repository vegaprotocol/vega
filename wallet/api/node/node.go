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

package node

import (
	"context"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/nodes_mocks.go -package mocks code.vegaprotocol.io/vega/wallet/api/node Node,Selector

// Node is the component used to get network information and send transactions.
type Node interface {
	Host() string
	Stop() error
	CheckTransaction(context.Context, *commandspb.Transaction) error
	SendTransaction(context.Context, *commandspb.Transaction, apipb.SubmitTransactionRequest_Type) (string, error)
	Statistics(ctx context.Context) (nodetypes.Statistics, error)
	LastBlock(context.Context) (nodetypes.LastBlock, error)
	SpamStatistics(ctx context.Context, pubKey string) (nodetypes.SpamStatistics, error)
}

// ReportType defines the type of event that occurred.
type ReportType string

var (
	InfoEvent    ReportType = "Info"
	WarningEvent ReportType = "Warning"
	ErrorEvent   ReportType = "Error"
	SuccessEvent ReportType = "Success"
)

type SelectionReporter func(ReportType, string)

// Selector implementing the strategy for node selection.
type Selector interface {
	Node(ctx context.Context, reporterFn SelectionReporter) (Node, error)
	Stop()
}
