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

package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

func toProtos[T proto.Message, V entities.ProtoEntity[T]](inputs []V) []T {
	protos := make([]T, 0, len(inputs))
	for _, input := range inputs {
		proto := input.ToProto()
		protos = append(protos, proto)
	}
	return protos
}

func mapSlice[T proto.Message, V any](inputs []V, toProto func(V) (T, error)) ([]T, error) {
	protos := make([]T, 0, len(inputs))
	for _, input := range inputs {
		proto, err := toProto(input)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to proto: %w", err)
		}
		protos = append(protos, proto)
	}
	return protos, nil
}

// queryProtoEntities invokes queryFunc and converts the entity to protos.
func queryProtoEntities[T proto.Message, E entities.ProtoEntity[T]](
	ctx context.Context, eg *errgroup.Group, txHash entities.TxHash,
	queryFunc func(ctx context.Context, txHash entities.TxHash) ([]E, error),
	apiErr error,
) chan []T {
	outChan := make(chan []T, 1)

	eg.Go(func() error {
		items, err := queryFunc(ctx, txHash)
		if err != nil {
			return apiError(codes.Internal, apiErr, err)
		}

		outChan <- toProtos[T](items)
		return nil
	})

	return outChan
}

type mapableEntities interface {
	entities.Entities | entities.LedgerEntry | entities.Transfer | entities.MarginLevels
}

// queryAndMapEntities invokes queryFunc and maps every single entity with mapFunc.
func queryAndMapEntities[T proto.Message, E mapableEntities](
	ctx context.Context, eg *errgroup.Group, txHash entities.TxHash,
	queryFunc func(context.Context, entities.TxHash) ([]E, error),
	mapFunc func(E) (T, error),
	apiErr error,
) chan []T {
	outChan := make(chan []T, 1)

	eg.Go(func() error {
		items, err := queryFunc(ctx, txHash)
		if err != nil {
			return apiError(codes.Internal, apiErr, err)
		}

		mapped, err := mapSlice(items, mapFunc)
		if err != nil {
			return apiError(codes.Internal, apiErr, err)
		}

		outChan <- mapped
		return nil
	})

	return outChan
}
