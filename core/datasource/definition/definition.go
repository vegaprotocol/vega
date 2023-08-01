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

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package definition

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	dserrors "code.vegaprotocol.io/vega/core/datasource/errors"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/internal/timetrigger"
	"code.vegaprotocol.io/vega/core/datasource/internal/vegatime"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type ContentType int32

const (
	ContentTypeInvalid ContentType = iota
	ContentTypeOracle
	ContentTypeEthOracle
	ContentTypeInternalTimeTermination
	ContentTypeInternalTimeTriggerTermination
)

type Definition struct {
	common.DataSourceType
}

func NewWith(dst common.DataSourceType) *Definition {
	if dst == nil {
		return &Definition{}
	}
	return &Definition{
		DataSourceType: dst.DeepClone(),
	}
}

// New creates a new EMPTY Definition object.
// TODO: eth oracle type too.
func New(tp ContentType) *Definition {
	ds := &Definition{}
	switch tp {
	case ContentTypeOracle:
		return NewWith(
			signedoracle.SpecConfiguration{
				Signers: []*common.Signer{},
				Filters: []*common.SpecFilter{},
			})
	case ContentTypeEthOracle:
		return NewWith(
			ethcallcommon.Spec{
				AbiJson:  []byte{},
				ArgsJson: []string{},
				Trigger:  &ethcallcommon.TimeTrigger{},
				Filters:  common.SpecFilters{},
			})
	case ContentTypeInternalTimeTermination:
		return NewWith(
			vegatime.SpecConfiguration{
				Conditions: []*common.SpecCondition{},
			})
	case ContentTypeInternalTimeTriggerTermination:
		return NewWith(
			timetrigger.SpecConfiguration{
				Triggers:   common.InternalTimeTriggers{},
				Conditions: []*common.SpecCondition{},
			})
	}
	return ds
}

// IntoProto returns the proto object from Definition
// that is - vegapb.DataSourceDefinition that may have external or internal SourceType.
// Returns the whole proto object.
func (s *Definition) IntoProto() *vegapb.DataSourceDefinition {
	if s.DataSourceType == nil {
		return &vegapb.DataSourceDefinition{}
	}
	proto, err := s.ToDefinitionProto()
	if err != nil {
		// TODO: bubble error
		return &vegapb.DataSourceDefinition{}
	}

	return proto
}

// DeepClone returns a clone of the Definition object.
func (s Definition) DeepClone() common.DataSourceType {
	if s.DataSourceType != nil {
		return &Definition{
			DataSourceType: s.DataSourceType.DeepClone(),
		}
	}
	return nil
}

func (s Definition) String() string {
	if s.DataSourceType != nil {
		return s.DataSourceType.String()
	}
	return ""
}

func (s *Definition) Content() interface{} {
	return s.DataSourceType
}

// FromProto tries to build the Definiition object
// from the given proto object.
func FromProto(protoConfig *vegapb.DataSourceDefinition, tm *time.Time) (common.DataSourceType, error) {
	if protoConfig != nil {
		data := protoConfig.Content()
		switch dtp := data.(type) {
		case *vegapb.DataSourceSpecConfiguration:
			return signedoracle.SpecConfigurationFromProto(dtp), nil

		case *vegapb.EthCallSpec:
			return ethcallcommon.SpecFromProto(dtp)

		case *vegapb.DataSourceSpecConfigurationTime:
			return vegatime.SpecConfigurationFromProto(dtp), nil
		case *vegapb.DataSourceSpecConfigurationTimeTrigger:
			return timetrigger.SpecConfigurationFromProto(dtp, tm)
		}
	}

	return &Definition{}, nil
}

// GetSigners tries to get the signers from the Definition if they exist.
func (s *Definition) GetSigners() []*common.Signer {
	signers := []*common.Signer{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case signedoracle.SpecConfiguration:
			signers = tp.Signers
		}
	}

	return signers
}

// GetFilters tries to get the filters from the Definition if they exist.
func (s *Definition) GetFilters() []*common.SpecFilter {
	filters := []*common.SpecFilter{}

	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case signedoracle.SpecConfiguration:
			filters = tp.Filters

		case ethcallcommon.Spec:
			filters = tp.Filters

		case vegatime.SpecConfiguration:
			// TODO: Fix this to use the same method as in the vegatime package (example: as below)
			// For the case the internal data source is time based
			// (as of OT https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed)
			// We add the filter key values manually to match a time based data source
			// Ensure only a single filter has been created, that holds the first condition
			if len(tp.Conditions) > 0 {
				filters = append(
					filters,
					&common.SpecFilter{
						Key: &common.SpecPropertyKey{
							Name: vegatime.VegaTimeKey,
							Type: datapb.PropertyKey_TYPE_TIMESTAMP,
						},
						Conditions: []*common.SpecCondition{
							tp.Conditions[0],
						},
					},
				)
			}

		case timetrigger.SpecConfiguration:
			sc := s.GetInternalTimeTriggerSpecConfiguration()
			filters = sc.GetFilters()
		}
	}

	return filters
}

// GetSignedOracleSpecConfiguration returns the base object - vega oracle SpecConfiguration
// from the Definition.
func (s *Definition) GetSignedOracleSpecConfiguration() signedoracle.SpecConfiguration {
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case signedoracle.SpecConfiguration:
			return tp
		}
	}

	return signedoracle.SpecConfiguration{}
}

// GetEthCallSpec returns the base object - EthCallSpec
// from the Definition.
func (s *Definition) GetEthCallSpec() ethcallcommon.Spec {
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case ethcallcommon.Spec:
			return tp
		}
	}

	return ethcallcommon.Spec{}
}

// Definition is also a `Timer`.
func (s *Definition) GetTimeTriggers() common.InternalTimeTriggers {
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case timetrigger.SpecConfiguration:
			return tp.GetTimeTriggers()
		}
	}

	return common.InternalTimeTriggers{}
}

func (s *Definition) IsTriggered(tm time.Time) bool {
	data := s.Content()
	if data != nil {
		switch tp := data.(type) {
		case timetrigger.SpecConfiguration:
			return tp.IsTriggered(tm)
		}
	}

	return false
}

// UpdateFilters updates the Definition Filters.
func (s *Definition) UpdateFilters(filters []*common.SpecFilter) error {
	fTypeCheck := map[*common.SpecFilter]struct{}{}
	fNameCheck := map[string]struct{}{}
	for _, f := range filters {
		if _, ok := fTypeCheck[f]; ok {
			return dserrors.ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList
		}
		if f.Key != nil {
			if _, ok := fNameCheck[f.Key.Name]; ok {
				return dserrors.ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList
			}
			fNameCheck[f.Key.Name] = struct{}{}
		}
		fTypeCheck[f] = struct{}{}
	}

	// maybe todo - enforce that it's never nil
	if s.DataSourceType == nil {
		return nil
	}

	switch content := s.DataSourceType.DeepClone().(type) {
	case signedoracle.SpecConfiguration:
		content.Filters = filters
		s.DataSourceType = content

	case ethcallcommon.Spec:
		content.Filters = filters
		s.DataSourceType = content

	case vegatime.SpecConfiguration:
		// The data source definition is an internal time based source
		// For this case we take only the first item from the list of filters
		// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
		c := []*common.SpecCondition{}
		if len(filters) > 0 {
			if len(filters[0].Conditions) > 0 {
				c = append(c, filters[0].Conditions[0])
			}
		}
		content.Conditions = c
		s.DataSourceType = content

	case timetrigger.SpecConfiguration:
		c := []*common.SpecCondition{}
		if len(filters) > 0 {
			for _, f := range filters {
				if len(f.Conditions) > 0 {
					c = append(c, f.Conditions...)
				}
			}
		}
		content.Conditions = c
		s.DataSourceType = content

	default:
		return fmt.Errorf("unable to set filters on data source type: %T", content)
	}
	return nil
}

func (s *Definition) SetFilterDecimals(d uint64) *Definition {
	switch content := s.DataSourceType.DeepClone().(type) {
	case signedoracle.SpecConfiguration:
		for i := range content.Filters {
			content.Filters[i].Key.NumberDecimalPlaces = &d
		}
		s.DataSourceType = content
	case ethcallcommon.Spec:
		for i := range content.Filters {
			content.Filters[i].Key.NumberDecimalPlaces = &d
		}
		s.DataSourceType = content

	default:
		// we should really be returning an error here but this method is only used in the integration tests
		panic(fmt.Sprintf("unable to set filter decimals on data source type: %T", content))
	}
	return s
}

// SetOracleConfig sets a given oracle config in the receiver.
// If the receiver is not external oracle type data source - it is not changed.
// This method does not care about object previous contents.
func (s *Definition) SetOracleConfig(oc common.DataSourceType) *Definition {
	if _, ok := s.DataSourceType.(signedoracle.SpecConfiguration); ok {
		s.DataSourceType = oc.DeepClone()
	}

	if _, ok := s.DataSourceType.(ethcallcommon.Spec); ok {
		s.DataSourceType = oc.DeepClone()
	}

	return s
}

// SetTimeTriggerConditionConfig sets a given conditions config in the receiver.
// If the receiver is not a time triggered data source - it does not set anything to it.
// This method does not care about object previous contents.
func (s *Definition) SetTimeTriggerConditionConfig(c []*common.SpecCondition) *Definition {
	if _, ok := s.DataSourceType.(vegatime.SpecConfiguration); ok {
		s.DataSourceType = vegatime.SpecConfiguration{
			Conditions: c,
		}
	}

	if sc, ok := s.DataSourceType.(timetrigger.SpecConfiguration); ok {
		s.DataSourceType = timetrigger.SpecConfiguration{
			Triggers:   sc.Triggers,
			Conditions: c,
		}
	}
	return s
}

func (s *Definition) SetTimeTriggerTriggersConfig(tr common.InternalTimeTriggers) *Definition {
	if sc, ok := s.DataSourceType.(timetrigger.SpecConfiguration); ok {
		s.DataSourceType = timetrigger.SpecConfiguration{
			Triggers:   tr,
			Conditions: sc.Conditions,
		}
	}
	return s
}

func (s *Definition) GetVegaTimeSpecConfiguration() vegatime.SpecConfiguration {
	data := s.Content()
	switch tp := data.(type) {
	case vegatime.SpecConfiguration:
		return tp
	}

	return vegatime.SpecConfiguration{}
}

func (s *Definition) GetInternalTimeTriggerSpecConfiguration() timetrigger.SpecConfiguration {
	data := s.Content()
	switch tp := data.(type) {
	case timetrigger.SpecConfiguration:
		return tp
	}
	return timetrigger.SpecConfiguration{}
}

func (s *Definition) IsExternal() (bool, error) {
	switch s.DataSourceType.(type) {
	case signedoracle.SpecConfiguration:
		return true, nil
	case ethcallcommon.Spec:
		return true, nil
	case vegatime.SpecConfiguration:
		return false, nil
	case timetrigger.SpecConfiguration:
		return false, nil
	}
	return false, errors.New("unknown type of data source provided")
}

func (s *Definition) Type() (ContentType, bool) {
	switch s.DataSourceType.(type) {
	case signedoracle.SpecConfiguration:
		return ContentTypeOracle, true
	case ethcallcommon.Spec:
		return ContentTypeEthOracle, true
	case vegatime.SpecConfiguration:
		return ContentTypeInternalTimeTermination, false
	case timetrigger.SpecConfiguration:
		return ContentTypeInternalTimeTriggerTermination, false
	}
	return ContentTypeInvalid, false
}

func (s *Definition) GetSpecConfiguration() common.DataSourceType {
	switch s.DataSourceType.(type) {
	case signedoracle.SpecConfiguration:
		return s.GetSignedOracleSpecConfiguration()
	case ethcallcommon.Spec:
		return s.GetEthCallSpec()
	case vegatime.SpecConfiguration:
		return s.GetVegaTimeSpecConfiguration()
	case timetrigger.SpecConfiguration:
		return s.GetInternalTimeTriggerSpecConfiguration()
	}

	return nil
}
