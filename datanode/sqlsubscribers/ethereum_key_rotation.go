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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
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

func (kr *EthereumKeyRotation) Name() string {
	return "EthereumKeyRotation"
}
