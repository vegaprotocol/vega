package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type EthCallSpec struct {
	Address               string
	Abi                   [][]byte // ?
	Method                string
	Args                  [][]byte
	Trigger               trigger
	RequiredConfirmations uint64
	Filter                *EthFilter
	Normaliser            normaliser
}

func (c *EthCallSpec) isDataSourceType() {}

func (s *EthCallSpec) oneOfProto() interface{} {
	return s.IntoProto()
}

func (s *EthCallSpec) IntoProto() *vegapb.EthCallSpec {
	ecs := &vegapb.EthCallSpec{}

	if s != nil {
		ecs.Address = s.Address
		if s.Abi != nil {

		}
		ecs.Method = s.Method

		if s.Args != nil {

		}

		if s.Trigger != nil {

		}

		ecs.RequiredConfirmations = s.RequiredConfirmations
		if s.Filter != nil {

		}

		if s.Normaliser != nil {

		}
	}
	return ecs
}

func (s *EthCallSpec) String() string {
	abi := ""
	for i, apos := range s.Abi {
		if i == 0 {
			abi = string(apos)
		} else {
			abi = abi + fmt.Sprintf(", %s", string(apos))
		}
	}

	args := ""
	for i, arg := range s.Abi {
		if i == 0 {
			args = string(arg)
		} else {
			args = args + fmt.Sprintf(", %s", string(arg))
		}
	}
	return fmt.Sprintf(
		"address(%s) abi(%s) method(%s) args(%s) trigger(%s) requiredConfirmations(%d) filter(%s) normaliser(%s)",
		s.Address,
		abi,
		s.Method,
		args,
		s.Trigger.String(),
		s.RequiredConfirmations,
		s.Filter.String(),
		s.Normaliser.String(),
	)
}

func (s *EthCallSpec) DeepClone() dataSourceType {
	return s // TODO: Check if this method is needed and fix it if it is.
}

func EthCallSpecFromProto(protoSpec *vegapb.DataSourceDefinitionExternal_EthOracle) *EthCallSpec {
	ethc := &EthCallSpec{}

	if protoSpec != nil {
		if protoSpec.EthOracle != nil {
			filters := []*DataSourceSpecFilter{}
			if protoSpec.EthOracle.Filter != nil {
				filters, _ = DataSourceSpecFiltersFromProto(protoSpec.EthOracle.Filter.Filters)
			}

			ethc.Filter = &EthFilter{
				Filters: filters,
			}
			abi := [][]byte{} // TODO: Handle ABI
			ethc.Address = protoSpec.EthOracle.Address
			ethc.Abi = abi
			ethc.Method = protoSpec.EthOracle.Method
			args := [][]byte{} // TODO: handle Args
			ethc.Args = args
			ethc.Trigger = EthCallTriggerFromProto(protoSpec.EthOracle.Trigger)
			ethc.RequiredConfirmations = protoSpec.EthOracle.RequiredConfirmations
			ethc.Filter, _ = EthFilterFromProto(protoSpec.EthOracle.Filter)
			norm, err := NormaliserFromProto(protoSpec.EthOracle.Normaliser)
			if err != nil {
				// What we do here? Return err across the whole path above?
			}
			ethc.Normaliser = norm

		}
	}

	return &EthCallSpec{} // TODO: Finish
}

type DataSourceDefinitionExternalEthOracle struct {
	EthOracle *EthCallSpec
}

func (e *DataSourceDefinitionExternalEthOracle) isDataSourceType() {}

func (e *DataSourceDefinitionExternalEthOracle) String() string {
	if e.EthOracle == nil {
		return ""
	}

	return e.EthOracle.String()
}

func (e *DataSourceDefinitionExternalEthOracle) IntoProto() *vegapb.DataSourceDefinitionExternal_EthOracle {
	eo := &vegapb.EthCallSpec{}

	if e.EthOracle != nil {
		eo = e.EthOracle.IntoProto()
	}

	return &vegapb.DataSourceDefinitionExternal_EthOracle{
		EthOracle: eo,
	}
}

func (e *DataSourceDefinitionExternalEthOracle) oneOfProto() interface{} {
	return e.IntoProto()
}
