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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/logging"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type OracleDataEvent interface {
	events.Event
	OracleData() oraclespb.OracleData
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_data_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/sqlsubscribers OracleDataStore
type OracleDataStore interface {
	Add(context.Context, *entities.OracleData) error
}

type OracleData struct {
	subscriber
	store OracleDataStore
	log   *logging.Logger
}

func NewOracleData(store OracleDataStore, log *logging.Logger) *OracleData {
	return &OracleData{
		store: store,
		log:   log,
	}
}

func (od *OracleData) Types() []events.Type {
	return []events.Type{events.OracleDataEvent}
}

func (od *OracleData) Push(ctx context.Context, evt events.Event) error {
	return od.consume(ctx, evt.(OracleDataEvent))
}

func (od *OracleData) consume(ctx context.Context, event OracleDataEvent) error {
	data := event.OracleData()
	entity, err := entities.OracleDataFromProto(&data, od.vegaTime)
	if err != nil {
		errors.Wrap(err, "converting oracle data proto to database entity failed")
	}

	return errors.Wrap(od.store.Add(ctx, entity), "inserting oracle data proto to SQL store failed")
}
