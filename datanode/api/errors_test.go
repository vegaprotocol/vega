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

package api_test

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/entities"
	types "code.vegaprotocol.io/vega/protos/vega"
)

func TestErrorMapUniqueCodes(t *testing.T) {
	errs := api.ErrorMap()
	existing := map[int32]bool{}
	for key, code := range errs {
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
		name        string
		args        args
		wantErr     assert.ErrorAssertionFunc
		wantCode    codes.Code
		wantStr     string
		wantMessage string
		wantDetails proto.Message
	}{
		{
			name: "nil error",
			args: args{
				err: nil,
				msg: []error{},
			},
			wantErr: assert.NoError,
		}, {
			name: "internal error",
			args: args{
				err: api.ErrOrderServiceGetOrders,
				msg: []error{errors.New("postgres has failed you")},
			},
			wantErr:     assert.Error,
			wantStr:     "rpc error: code = Internal desc = Internal error",
			wantCode:    codes.Internal,
			wantMessage: "Internal error",
			wantDetails: &types.ErrorDetail{
				Code:    20007,
				Message: "failed to get orders",
				Inner:   "postgres has failed you",
			},
		}, {
			name: "invalid arguments error",
			args: args{
				err: api.ErrMissingProposalID,
			},
			wantErr:     assert.Error,
			wantStr:     "rpc error: code = InvalidArgument desc = InvalidArgument error",
			wantCode:    codes.InvalidArgument,
			wantMessage: "InvalidArgument error",
			wantDetails: &types.ErrorDetail{
				Code:    10021,
				Message: "proposal id is a required parameter",
			},
		}, {
			name: "not found error",
			args: args{
				err: api.ErrOrderNotFound,
				msg: []error{entities.ErrNotFound},
			},
			wantErr:     assert.Error,
			wantStr:     "rpc error: code = NotFound desc = NotFound error",
			wantCode:    codes.NotFound,
			wantMessage: "NotFound error",
			wantDetails: &types.ErrorDetail{
				Code:    20006,
				Message: "order not found",
				Inner:   "no resource corresponding to this id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := api.FormatE(tt.args.err, tt.args.msg...)
			tt.wantErr(t, err, "formatE()")
			if len(tt.wantStr) > 0 {
				assert.EqualError(t, err, tt.wantStr, "formatE()")
				s, ok := status.FromError(err)
				assert.True(t, ok, "FromError()")
				assert.Equal(t, tt.wantCode, s.Code(), "Code")
				assert.Equal(t, tt.wantMessage, s.Message(), "Message")
				require.Len(t, s.Details(), 1)
				d, ok := s.Details()[0].(proto.Message)
				require.True(t, ok)
				if !proto.Equal(tt.wantDetails, d) {
					t.Errorf("Details are not the same:\n\twant: %v\n\t got: %v", tt.wantDetails, d)
				}
			}
		})
	}
}
