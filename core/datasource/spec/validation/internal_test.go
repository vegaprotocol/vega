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

package validation_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/datasource/spec/validation"

	"github.com/stretchr/testify/require"
)

func TestCheckForInternalOracle(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should return an error if there is any data that contains reserved prefix",
			args: args{
				data: map[string]string{
					"aaaa":                      "aaaa",
					"bbbb":                      "bbbb",
					"cccc":                      "cccc",
					"vegaprotocol.builtin.dddd": "dddd",
				},
			},
			wantErr: true,
		},
		{
			name: "Should pass validation if none of the data contains a reserved prefix",
			args: args{
				data: map[string]string{
					"aaaa": "aaaa",
					"bbbb": "bbbb",
					"cccc": "cccc",
					"dddd": "dddd",
				},
			},
			wantErr: false,
		},
		{
			name: "Should pass validation if reserved prefix is contained in key, but key doesn't start with the prefix",
			args: args{
				data: map[string]string{
					"aaaa":                      "aaaa",
					"bbbb":                      "bbbb",
					"cccc":                      "cccc",
					"dddd.vegaprotocol.builtin": "dddd",
				},
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			if err := validation.CheckForInternalOracle(tc.args.data); tc.wantErr {
				require.Error(tt, err)
			} else {
				require.NoError(tt, err)
			}
		})
	}
}
