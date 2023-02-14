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

package sqlsubscribers

import (
	"context"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type EthereumKeyRotationEvent interface {
	events.Event
	EthereumKeyRotation() eventspb.EthereumKeyRotation
}

type EthereumKeyRotationService interface {
	Add(context.Context, entities.EthereumKeyRotation) error
}

type EthereumKeyRotation struct {
	subscriber
	service EthereumKeyRotationService
}

func NewEthereumKeyRotation(service EthereumKeyRotationService) *EthereumKeyRotation {
	return &EthereumKeyRotation{
		service: service,
	}
}

func (kr *EthereumKeyRotation) Types() []events.Type {
	return []events.Type{events.EthereumKeyRotationEvent}
}

func (kr *EthereumKeyRotation) Push(ctx context.Context, evt events.Event) error {
	return kr.consume(ctx, evt.(EthereumKeyRotationEvent))
}

func (kr *EthereumKeyRotation) consume(ctx context.Context, event EthereumKeyRotationEvent) error {
	keyRotation := event.EthereumKeyRotation()
	record, err := entities.EthereumKeyRotationFromProto(&keyRotation, entities.TxHash(event.TxHash()), kr.vegaTime,
		event.Sequence())
	if err != nil {
		return errors.Wrap(err, "converting ethereum key rotation proto to database entity failed")
	}

	return errors.Wrap(kr.service.Add(ctx, record), "Inserting ethereum key rotation to SQL store failed")
}
