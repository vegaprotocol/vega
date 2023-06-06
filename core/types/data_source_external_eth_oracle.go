package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type EthCallSpec struct {
	Address               string
	AbiJson               string
	Method                string
	ArgsJson              []string
	Trigger               *EthCallTrigger
	RequiredConfirmations uint64
	Filter                *EthFilter
	Normaliser            normaliser
}

// Why not just an id?
func (s *EthCallSpec) HashHex() string {
	hashFunc := sha3.New256()

	hashFunc.Write([]byte(s.Address))
	hashFunc.Write([]byte(s.Method))
	hashFunc.Write([]byte(strings.Join(s.ArgsJson, ",")))
	hashFunc.Write([]byte(s.AbiJson))
	hashFunc.Write(s.Trigger.EthTrigger.Hash())
	hashFunc.Write(s.Filter.Hash())
	hashFunc.Write([]byte(fmt.Sprintf("requiredconfirmations: %v", s.RequiredConfirmations)))

	return hex.EncodeToString(hashFunc.Sum(nil))
}

func (s *EthCallSpec) isDataSourceType() {}

func (s *EthCallSpec) oneOfProto() interface{} {
	return s.IntoProto()
}

func (s *EthCallSpec) IntoProto() *vegapb.EthCallSpec {
	ecs := &vegapb.EthCallSpec{}
	/*
		if s != nil {
			ecs.Address = s.Address
			if s.AbiJson != "" {
			}
			ecs.Method = s.Method

			if s.ArgsJson != nil {
			}

			if s.Trigger != nil {
			}

			ecs.RequiredConfirmations = s.RequiredConfirmations
			if s.Filter != nil {
			}

			if s.Normaliser != nil {
			}
		}*/
	return ecs
}

func (s *EthCallSpec) String() string {
	abi := ""
	for i, apos := range s.AbiJson {
		if i == 0 {
			abi = string(apos)
		} else {
			abi = abi + fmt.Sprintf(", %s", string(apos))
		}
	}

	args := ""
	for i, arg := range s.AbiJson {
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
		s.Trigger.EthTrigger.String(),
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

			ethc.Address = protoSpec.EthOracle.Address
			abiBytes, err := protoSpec.EthOracle.Abi.MarshalJSON()
			if err != nil {
				// TODO: Handle error - pass this up the stack?
				panic(err)
			}
			ethc.AbiJson = string(abiBytes)

			ethc.Method = protoSpec.EthOracle.Method
			jsonArgs := []string{}
			for _, protoArg := range protoSpec.EthOracle.Args {
				jsonArg, err := protoArg.MarshalJSON()
				if err != nil {
					// TODO: Handle error - pass this up the stack?
					panic(err)
				}
				jsonArgs = append(jsonArgs, string(jsonArg))
			}

			ethc.ArgsJson = jsonArgs
			ethc.Trigger = EthCallTriggerFromSpec(protoSpec.EthOracle.Trigger)
			ethc.RequiredConfirmations = protoSpec.EthOracle.RequiredConfirmations
			ethc.Filter, _ = EthFilterFromProto(protoSpec.EthOracle.Filter)
			norm := NormaliserFromProto(protoSpec.EthOracle.Normaliser)
			ethc.Normaliser = norm
		}
	}

	return ethc
}

type DataSourceDefinitionExternalEthOracle struct {
	EthOracle *EthCallSpec
}

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
