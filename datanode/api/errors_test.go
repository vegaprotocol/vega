// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMapUniqueCodes(t *testing.T) {
	errors := ErrorMap()
	existing := map[int32]bool{}
	for key, code := range errors {
		if _, ok := existing[code]; ok {
			t.Log("Duplicate code found in api.ErrorMap for code, duplicate =>", code, key)
			t.Fail()
			return
		}
		existing[code] = true
	}
}

func Test_formatE(t *testing.T) {
	type args struct {
		err error
		msg []error
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
		want    string
	}{
		{
			name: "nil error",
			args: args{
				err: nil,
				msg: []error{errors.New("test")},
			},
			wantErr: assert.NoError,
		}, {
			name: "non-nil error",
			args: args{
				err: fmt.Errorf("test"),
				msg: []error{errors.New("test")},
			},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
				return assert.Error(t, err, msgAndArgs...)
			},
		}, {
			name: "invalid arguments error",
			args: args{
				err: ErrMissingProposalID,
				msg: []error{errors.New("test")},
			},
			wantErr: assert.Error,
			want:    "rpc error: code = InvalidArgument desc = InvalidArgument error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatE(tt.args.err, tt.args.msg...)
			tt.wantErr(t, err, "formatE()")
			if len(tt.want) > 0 {
				assert.EqualError(t, err, tt.want, "formatE()")
			}
		})
	}
}
