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

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestUpdatePartyProfile(t *testing.T) {
	t.Run("Updating party's profile succeeds", testUpdatePartyProfileSucceeds)
	t.Run("Updating party's profile with invalid alias fails", testUpdatePartyProfileWithInvalidAliasFails)
	t.Run("Updating party's profile with invalid metadata fails", testUpdatePartyProfileWithInvalidMetadataFails)
}

func testUpdatePartyProfileSucceeds(t *testing.T) {
	tcs := []struct {
		name string
		cmd  *commandspb.UpdatePartyProfile
	}{
		{
			name: "when empty",
			cmd:  &commandspb.UpdatePartyProfile{},
		}, {
			name: "with an alias",
			cmd: &commandspb.UpdatePartyProfile{
				Alias: "test",
			},
		}, {
			name: "with metadata",
			cmd: &commandspb.UpdatePartyProfile{
				Metadata: []*vegapb.Metadata{
					{
						Key:   "key",
						Value: "value",
					},
				},
			},
		}, {
			name: "with both",
			cmd: &commandspb.UpdatePartyProfile{
				Alias: "test",
				Metadata: []*vegapb.Metadata{
					{
						Key:   "key",
						Value: "value",
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := checkUpdatePartyProfile(t, tc.cmd)

			assert.Empty(t, err)
		})
	}
}

// 0088-PPRF-001.
func testUpdatePartyProfileWithInvalidAliasFails(t *testing.T) {
	tcs := []struct {
		name  string
		alias string
		err   error
	}{
		{
			name:  "with more than 32 characters",
			alias: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			err:   commands.ErrIsLimitedTo32Characters,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := checkUpdatePartyProfile(t, &commandspb.UpdatePartyProfile{
				Alias: tc.alias,
			})

			assert.Contains(t, err.Get("update_party_profile.alias"), tc.err)
		})
	}
}

// 0088-PPRF-003, 0088-PPRF-004, 0088-PPRF-005.
func testUpdatePartyProfileWithInvalidMetadataFails(t *testing.T) {
	tcs := []struct {
		name     string
		metadata []*vegapb.Metadata
		field    string
		err      error
	}{
		{
			name:     "with more than 10 entries",
			metadata: []*vegapb.Metadata{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
			field:    "update_party_profile.metadata",
			err:      commands.ErrIsLimitedTo10Entries,
		}, {
			name: "with empty key",
			metadata: []*vegapb.Metadata{
				{
					Key:   "",
					Value: "",
				},
			},
			field: "update_party_profile.metadata.0.key",
			err:   commands.ErrCannotBeBlank,
		}, {
			name: "with key more than 32 characters",
			metadata: []*vegapb.Metadata{
				{
					Key:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					Value: "",
				},
			},
			field: "update_party_profile.metadata.0.key",
			err:   commands.ErrIsLimitedTo32Characters,
		}, {
			name: "with duplicated key",
			metadata: []*vegapb.Metadata{
				{
					Key:   "hello",
					Value: "value1",
				}, {
					Key:   "hello",
					Value: "value2",
				},
			},
			field: "update_party_profile.metadata.1.key",
			err:   commands.ErrIsDuplicated,
		}, {
			name: "with value more than 255 characters",
			metadata: []*vegapb.Metadata{
				{
					Key: "test",
					Value: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			},
			field: "update_party_profile.metadata.0.value",
			err:   commands.ErrIsLimitedTo255Characters,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := checkUpdatePartyProfile(t, &commandspb.UpdatePartyProfile{
				Metadata: tc.metadata,
			})

			assert.Contains(t, err.Get(tc.field), tc.err)
		})
	}
}

func checkUpdatePartyProfile(t *testing.T, cmd *commandspb.UpdatePartyProfile) commands.Errors {
	t.Helper()

	err := commands.CheckUpdatePartyProfile(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
