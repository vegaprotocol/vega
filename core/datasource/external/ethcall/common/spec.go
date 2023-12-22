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

package common

import (
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/datasource/common"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrCallSpecIsNil      = errors.New("ethereum call spec proto is nil")
	ErrInvalidEthereumAbi = errors.New("is not a valid ethereum abi definition")
	ErrInvalidCallTrigger = errors.New("ethereum call trigger not valid")
	ErrInvalidCallArgs    = errors.New("ethereum call args not valid")
	ErrInvalidFilters     = errors.New("ethereum call filters not valid")
)

type Spec struct {
	Address               string
	AbiJson               []byte
	Method                string
	ArgsJson              []string
	Trigger               Trigger
	RequiredConfirmations uint64
	Normalisers           map[string]string
	Filters               common.SpecFilters
	L2ChainID             uint64
}

func SpecFromProto(proto *vegapb.EthCallSpec) (Spec, error) {
	if proto == nil {
		return Spec{}, ErrCallSpecIsNil
	}

	trigger, err := TriggerFromProto(proto.Trigger)
	if err != nil {
		return Spec{}, errors.Join(ErrInvalidCallTrigger, err)
	}

	filters := common.SpecFiltersFromProto(proto.Filters)

	abiBytes := []byte(proto.Abi)

	jsonArgs := []string{}
	for _, protoArg := range proto.Args {
		jsonArg, err := protoArg.MarshalJSON()
		if err != nil {
			return Spec{}, errors.Join(ErrInvalidCallArgs, err)
		}
		jsonArgs = append(jsonArgs, string(jsonArg))
	}

	normalisers := map[string]string{}
	for _, v := range proto.Normalisers {
		normalisers[v.Name] = v.Expression
	}

	// default to ethereum mainnet
	var chainID uint64 = 1
	if proto.L2ChainId != nil {
		chainID = *proto.L2ChainId
	}

	return Spec{
		Address:               proto.Address,
		AbiJson:               abiBytes,
		Method:                proto.Method,
		ArgsJson:              jsonArgs,
		Trigger:               trigger,
		RequiredConfirmations: proto.RequiredConfirmations,
		Filters:               filters,
		Normalisers:           normalisers,
		L2ChainID:             chainID,
	}, nil
}

func (s Spec) IntoProto() (*vegapb.EthCallSpec, error) {
	argsPBValue := []*structpb.Value{}
	for _, arg := range s.ArgsJson {
		v := structpb.Value{}
		err := v.UnmarshalJSON([]byte(arg))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal arg json '%s': %w", arg, err)
		}
		argsPBValue = append(argsPBValue, &v)
	}

	normalisers := []*vegapb.Normaliser{}
	for k, v := range s.Normalisers {
		n := vegapb.Normaliser{
			Name:       k,
			Expression: v,
		}
		normalisers = append(normalisers, &n)
	}

	sort.Slice(normalisers, func(i, j int) bool { return normalisers[i].Name < normalisers[j].Name })

	return &vegapb.EthCallSpec{
		Address:               s.Address,
		Abi:                   string(s.AbiJson),
		Method:                s.Method,
		Args:                  argsPBValue,
		Trigger:               s.Trigger.IntoTriggerProto(),
		RequiredConfirmations: s.RequiredConfirmations,
		Filters:               s.Filters.IntoProto(),
		Normalisers:           normalisers,
	}, nil
}

func (s Spec) ToDefinitionProto() (*vegapb.DataSourceDefinition, error) {
	eth, err := s.IntoProto()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to eth oracle proto: %w", err)
	}

	return &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_External{
			External: &vegapb.DataSourceDefinitionExternal{
				SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
					EthOracle: eth,
				},
			},
		},
	}, nil
}

func (s Spec) GetFilters() []*common.SpecFilter {
	return s.Filters
}

func (s Spec) String() string {
	return fmt.Sprintf("ethcallspec(%v, %v, %v, %v, %v, %v, %v)",
		s.Address, s.AbiJson, s.Method, s.ArgsJson, s.Trigger, s.RequiredConfirmations, s.Filters)
}

// Whats the need for this deep clone?
func (s Spec) DeepClone() common.DataSourceType {
	clonedNormalisers := make(map[string]string)
	for key, value := range s.Normalisers {
		clonedNormalisers[key] = value
	}

	return Spec{
		Address:               s.Address,
		AbiJson:               s.AbiJson,
		Method:                s.Method,
		ArgsJson:              append([]string(nil), s.ArgsJson...),
		Trigger:               s.Trigger,
		RequiredConfirmations: s.RequiredConfirmations,
		Filters:               append(common.SpecFilters(nil), s.Filters...),
		Normalisers:           clonedNormalisers,
	}
}

func (s Spec) IsZero() bool {
	return s.Address == "" &&
		s.Method == "" &&
		s.Trigger == nil &&
		s.RequiredConfirmations == 0 &&
		len(s.AbiJson) == 0 &&
		len(s.ArgsJson) == 0 &&
		len(s.Filters) == 0 &&
		len(s.Normalisers) == 0
}
