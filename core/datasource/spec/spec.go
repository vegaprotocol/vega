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

package spec

import (
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type SpecID string

type Spec struct {
	// id is a unique identifier for the Spec
	id SpecID

	// signers list all the authorized public keys from where a Data can
	// come from.
	signers map[string]struct{}

	// any time triggers on the spec
	triggers common.InternalTimeTriggers

	// filters holds all the expected property keys with the conditions they
	// should match.
	filters common.Filters
	// OriginalSpec is the protobuf description of Spec
	OriginalSpec *datasource.Spec
}

// New builds a new Spec from a common.Spec (currently uses one level below - common.ExternalDataSourceSpec) in a form that
// suits the processing of the filters.
// Spec allows the existence of one and only one.
// Currently VEGA network utilises internal triggers in the oracle function path, even though
// the oracles are treated as external data sources.
// For this reason this function checks if the provided external type of data source definition
// contains a key name that indicates a builtin type of logic
// and if the given data source definition is an internal type of data source, for more context refer to
// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
func New(originalSpec datasource.Spec) (*Spec, error) {
	filtersFromSpec := []*common.SpecFilter{}
	signersFromSpec := []*common.Signer{}
	var triggersFromSpec common.InternalTimeTriggers

	isExtType := false
	var err error
	// if originalSpec != nil {
	if originalSpec.Data != nil {
		filtersFromSpec = originalSpec.Data.GetFilters()
		isExtType, err = originalSpec.Data.IsExternal()
		if err != nil {
			return nil, err
		}
	}
	//}

	builtInKey := false
	for _, f := range filtersFromSpec {
		if isExtType {
			if strings.HasPrefix(f.Key.Name, "vegaprotocol.builtin") && f.Key.Type == datapb.PropertyKey_TYPE_TIMESTAMP {
				builtInKey = true
			}
		}
	}

	builtInTrigger := false
	for _, f := range filtersFromSpec {
		if strings.HasPrefix(f.Key.Name, "vegaprotocol.builtin.timetrigger") && f.Key.Type == datapb.PropertyKey_TYPE_TIMESTAMP {
			builtInTrigger = true
		}
	}

	typedFilters, err := common.NewFilters(filtersFromSpec, isExtType)
	if err != nil {
		return nil, err
	}
	// We check if the filters list is empty in the proposal submission step.
	// We do not need to double that logic here.

	signers := map[string]struct{}{}
	if !builtInTrigger && !builtInKey && isExtType {
		// if originalSpec != nil {
		if originalSpec.Data != nil {
			src := *originalSpec.Data

			signersFromSpec = src.GetSigners()
		}
		//}

		// We check if the signers list is empty h in the proposal submission step.
		// We do not need to duble that logic here.

		for _, pk := range signersFromSpec {
			signers[pk.String()] = struct{}{}
		}
	}

	if builtInTrigger {
		gt := originalSpec.Data.GetTimeTriggers()
		triggersFromSpec = *gt
	}

	os := &Spec{
		id:           SpecID(originalSpec.ID),
		signers:      signers,
		filters:      typedFilters,
		triggers:     triggersFromSpec,
		OriginalSpec: &originalSpec,
	}

	return os, nil
}

func (s Spec) EnsureBoundableProperty(property string, propType datapb.PropertyKey_Type) error {
	return s.filters.EnsureBoundableProperty(property, propType)
}

func isInternalData(data common.Data) bool {
	for k := range data.Data {
		if !strings.HasPrefix(k, BuiltinPrefix) {
			return false
		}
	}

	return true
}

func isInternalTimeTrigger(data common.Data) (bool, time.Time) {
	for k, v := range data.Data {
		if k == BuiltinTimeTrigger {
			// convert string to time
			if t, err := strconv.ParseInt(v, 10, 0); err == nil {
				return true, time.Unix(t, 0)
			}
		}
	}
	return false, time.Time{}
}

// MatchSigners tries to match the public keys from the provided Data object with the ones
// present in the Spec.
func (s *Spec) MatchSigners(data common.Data) bool {
	return containsRequiredSigners(data.Signers, s.signers)
}

// MatchData indicates if a given Data matches the spec or not.
func (s *Spec) MatchData(data common.Data) (bool, error) {
	// if the data contains the internal source timestamp key, and only that key,
	// then we do not need to verify the public keys as there will not be one

	if !isInternalData(data) && !containsRequiredSigners(data.Signers, s.signers) {
		return false, nil
	}

	// Don't broadcast ethcall data based unless it's 'EthKey' matches
	// (which is currently the SpecID - see comment on the datasource.common.Data struct)
	if data.EthKey != "" && data.EthKey != string(s.id) {
		return false, nil
	}

	// if it is internal time data and we have a time-trigger check that we're past it
	if ok, tt := isInternalTimeTrigger(data); ok && s.triggers[0] != nil {
		if !s.triggers.IsTriggered(tt) {
			return false, nil
		}
	}

	return s.filters.Match(data.Data)
}

// containsRequiredSigners verifies if all the public keys in the Data
// are within the list of currently authorized by the Spec.
func containsRequiredSigners(dataSigners []*common.Signer, authPks map[string]struct{}) bool {
	for _, signer := range dataSigners {
		if _, ok := authPks[signer.String()]; !ok {
			return false
		}
	}
	return true
}
