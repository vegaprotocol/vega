// Copyright (c) 2022 Gobalsky Labs Limited
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

package oracles

import (
	"errors"
	"strings"

	"code.vegaprotocol.io/vega/core/oracles/filters"
	"code.vegaprotocol.io/vega/core/types"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

var (
	// ErrMissingSigners is returned when the datapb.OracleSpec is missing
	// its signers.
	ErrMissingSigners = errors.New("signers are required")
	// ErrAtLeastOneFilterIsRequired is returned when the datapb.OracleSpec
	// has no expected properties nor filters. At least one of these should be
	// defined.
	ErrAtLeastOneFilterIsRequired = errors.New("at least one filter is required")

	// ErrMissingPropertyName is returned when a property as no name.
	ErrMissingPropertyName = errors.New("a property name is required")
	// ErrInvalidPropertyKey is returned if validation finds a reserved Vega property key.
	ErrInvalidPropertyKey = errors.New("property key is reserved")
)

type OracleSpecID string

type OracleSpec struct {
	// id is a unique identifier for the OracleSpec
	id OracleSpecID

	// signers list all the authorized public keys from where an OracleData can
	// come from.
	signers map[string]struct{}

	// filters holds all the expected property keys with the conditions they
	// should match.
	filters filters.Filters
	// OriginalSpec is the protobuf description of OracleSpec
	OriginalSpec *types.OracleSpec
}

// NewOracleSpec builds an OracleSpec from a types.OracleSpec (currently uses one level below - types.ExternalDataSourceSpec) in a form that
// suits the processing of the filters.
// OracleSpec allows the existence of one and only one.
// Currently VEGA network utilises internal triggers in the oracle function path, even though
// the oracles are treated as external data sources.
// For this reason this function checks if the provided external type of data source definition
// contains a key name that indicates a builtin type of logic
// and if the given data source definition is an internal type of data source, for more context refer to
// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
func NewOracleSpec(originalSpec types.ExternalDataSourceSpec) (*OracleSpec, error) {
	filtersFromSpec := []*types.DataSourceSpecFilter{}
	signersFromSpec := []*types.Signer{}

	isExtType := false
	var err error
	if originalSpec.Spec != nil {
		if originalSpec.Spec.Data != nil {
			filtersFromSpec = originalSpec.Spec.Data.GetFilters()
			isExtType, err = originalSpec.Spec.Data.IsExternal()
			if err != nil {
				return nil, err
			}
		}
	}

	builtInKey := false
	for _, f := range filtersFromSpec {
		if isExtType {
			if strings.HasPrefix(f.Key.Name, "vegaprotocol.builtin") && f.Key.Type == datapb.PropertyKey_TYPE_TIMESTAMP {
				builtInKey = true
			}
		}
	}

	typedFilters, err := filters.NewFilters(filtersFromSpec, isExtType)
	if err != nil {
		return nil, err
	}
	// We check if the filters list is empty in the proposal submission step.
	// We do not need to double that logic here.

	signers := map[string]struct{}{}
	if !builtInKey && isExtType {
		if originalSpec.Spec != nil {
			if originalSpec.Spec.Data != nil {
				src := *originalSpec.Spec.Data

				signersFromSpec = src.GetSigners()
			}
		}

		// We check if the signers list is empty h in the proposal submission step.
		// We do not need to duble that logic here.

		for _, pk := range signersFromSpec {
			signers[pk.String()] = struct{}{}
		}
	}

	os := &OracleSpec{
		id:      OracleSpecID(originalSpec.Spec.ID),
		signers: signers,
		filters: typedFilters,
		OriginalSpec: &types.OracleSpec{
			ExternalDataSourceSpec: &originalSpec,
		},
	}

	return os, nil
}

func (s OracleSpec) EnsureBoundableProperty(property string, propType datapb.PropertyKey_Type) error {
	return s.filters.EnsureBoundableProperty(property, propType)
}

func isInternalOracleData(data OracleData) bool {
	for k := range data.Data {
		if !strings.HasPrefix(k, BuiltinOraclePrefix) {
			return false
		}
	}

	return true
}

// MatchSigners tries to match the public keys from the provided OracleData object with the ones
// present in the Spec.
func (s *OracleSpec) MatchSigners(data OracleData) bool {
	return containsRequiredSigners(data.Signers, s.signers)
}

// MatchData indicates if a given OracleData matches the spec or not.
func (s *OracleSpec) MatchData(data OracleData) (bool, error) {
	// if the data contains the internal oracle timestamp key, and only that key,
	// then we do not need to verify the public keys as there will not be one

	if !isInternalOracleData(data) && !containsRequiredSigners(data.Signers, s.signers) {
		return false, nil
	}

	return s.filters.Match(data.Data)
}

// containsRequiredSigners verifies if all the public keys in the OracleData
// are within the list of currently authorized by the OracleSpec.
func containsRequiredSigners(dataSigners []*types.Signer, authPks map[string]struct{}) bool {
	for _, signer := range dataSigners {
		if _, ok := authPks[signer.String()]; !ok {
			return false
		}
	}
	return true
}
