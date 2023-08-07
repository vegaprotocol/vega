// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package common

import (
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/datasource/common"
	verrors "code.vegaprotocol.io/vega/libs/errors"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrCallSpecIsNil      = errors.New("ethereum call spec proto is nil")
	ErrInvalidEthereumAbi = errors.New("is not a valid ethereum address")
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
}

func SpecFromProto(proto *vegapb.EthCallSpec) (Spec, error) {
	if proto == nil {
		return Spec{}, ErrCallSpecIsNil
	}

	trigger, err := TriggerFromProto(proto.Trigger)
	if err != nil {
		return Spec{}, verrors.Join(ErrInvalidCallTrigger, err)
	}

	filters := common.SpecFiltersFromProto(proto.Filters)

	abiBytes := []byte(proto.Abi)

	jsonArgs := []string{}
	for _, protoArg := range proto.Args {
		jsonArg, err := protoArg.MarshalJSON()
		if err != nil {
			return Spec{}, verrors.Join(ErrInvalidCallArgs, err)
		}
		jsonArgs = append(jsonArgs, string(jsonArg))
	}

	normalisers := map[string]string{}
	for _, v := range proto.Normalisers {
		normalisers[v.Name] = v.Expression
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
