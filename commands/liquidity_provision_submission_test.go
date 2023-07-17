package commands_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestNilLiquidityProvisionSubmissionFails(t *testing.T) {
	err := commands.CheckLiquidityProvisionSubmission(nil)

	assert.Error(t, err)
}

func TestLiquidityProvisionSubmission(t *testing.T) {
	cases := []struct {
		lp        commandspb.LiquidityProvisionSubmission
		errString string
	}{
		{
			// this is a valid cancellation.
			lp: commandspb.LiquidityProvisionSubmission{
				CommitmentAmount: "0",
				MarketId:         "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "liquidity_provision_submission.commitment_amount (is not a valid number)",
		},
		{
			lp: commandspb.LiquidityProvisionSubmission{
				CommitmentAmount: "100",
				Fee:              "abcd",
				MarketId:         "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "liquidity_provision_submission.fee (is not a valid value)",
		},
		{
			lp: commandspb.LiquidityProvisionSubmission{
				CommitmentAmount: "100",
				Fee:              "-1",
				MarketId:         "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "liquidity_provision_submission.fee (must be positive)",
		},
		{
			lp: commandspb.LiquidityProvisionSubmission{
				CommitmentAmount: "100",
				Fee:              "0.1",
			},
			errString: "liquidity_provision_submission.market_id (is required)",
		},

		{
			lp: commandspb.LiquidityProvisionSubmission{
				CommitmentAmount: "100",
				Fee:              "0.1",
				MarketId:         "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
		},
	}

	for i, c := range cases {
		err := commands.CheckLiquidityProvisionSubmission(&c.lp)
		if len(c.errString) <= 0 {
			assert.NoErrorf(t, err, "unexpected error on position: %d", i)
			continue
		}

		assert.Errorf(t, err, "expected error on position: %d", i)
		assert.EqualErrorf(t, err, c.errString, "expected error to match on position: %d", i)
	}
}

func TestCheckLiquidityProvisionCancellation(t *testing.T) {
	type args struct {
		cmd *commandspb.LiquidityProvisionCancellation
	}
	tests := []struct {
		name      string
		args      args
		wantErr   assert.ErrorAssertionFunc
		errString string
	}{
		{
			name: "Should return an error if request is nil",
			args: args{
				cmd: nil,
			},
			wantErr:   assert.Error,
			errString: "liquidity_provision_cancellation (is required)",
		},
		{
			name: "Should return an error if market_id is not provided",
			args: args{
				cmd: &commandspb.LiquidityProvisionCancellation{
					MarketId: "",
				},
			},
			wantErr:   assert.Error,
			errString: "liquidity_provision_cancellation.market_id (is required)",
		},
		{
			name: "Should succeed if market id is provided",
			args: args{
				cmd: &commandspb.LiquidityProvisionCancellation{
					MarketId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := commands.CheckLiquidityProvisionCancellation(tt.args.cmd)
			tt.wantErr(t, gotErr, fmt.Sprintf("CheckLiquidityProvisionCancellation(%v)", tt.args.cmd))
			if tt.errString != "" {
				assert.EqualError(t, gotErr, tt.errString)
			}
		})
	}
}

func TestCheckLiquidityProvisionAmendment(t *testing.T) {
	type args struct {
		cmd *commandspb.LiquidityProvisionAmendment
	}
	tests := []struct {
		name      string
		args      args
		wantErr   assert.ErrorAssertionFunc
		errString string
	}{
		{
			name: "Should return an error when the command is nil",
			args: args{
				cmd: nil,
			},
			wantErr:   assert.Error,
			errString: "liquidity_provision_amendment (is required)",
		},
		{
			name: "Should return an error when market_id is not provided",
			args: args{
				cmd: &commandspb.LiquidityProvisionAmendment{
					MarketId: "",
					Sells: []*types.LiquidityOrder{
						{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: "10", Proportion: 1},
					},
					Buys: []*types.LiquidityOrder{
						{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: "10", Proportion: 1},
					},
				},
			},
			wantErr:   assert.Error,
			errString: "liquidity_provision_amendment.market_id (is required)",
		},
		{
			name: "Should return an error if amendment changes nothing",
			args: args{
				cmd: &commandspb.LiquidityProvisionAmendment{
					MarketId: "abcd",
				},
			},
			wantErr:   assert.Error,
			errString: "liquidity_provision_amendment (is required)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := commands.CheckLiquidityProvisionAmendment(tt.args.cmd)
			tt.wantErr(t, gotErr, fmt.Sprintf("CheckLiquidityProvisionAmendment(%v)", tt.args.cmd))

			if tt.errString != "" {
				assert.EqualError(t, gotErr, tt.errString)
			}
		})
	}
}
