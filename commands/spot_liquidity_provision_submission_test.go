package commands_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestNilSpotLiquidityProvisionSubmissionFails(t *testing.T) {
	err := commands.CheckLiquidityProvisionSubmission(nil)

	assert.Error(t, err)
}

func TestSpotLiquidityProvisionSubmission(t *testing.T) {
	cases := []struct {
		lp        commandspb.SpotLiquidityProvisionSubmission
		errString string
	}{
		{
			lp: commandspb.SpotLiquidityProvisionSubmission{
				BuyCommitmentAmount:  "0",
				SellCommitmentAmount: "10",
				MarketId:             "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "spot_liquidity_provision_submission.buy_commitment_amount (is not a valid number)",
		},
		{
			lp: commandspb.SpotLiquidityProvisionSubmission{
				BuyCommitmentAmount:  "-10",
				SellCommitmentAmount: "10",
				MarketId:             "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "spot_liquidity_provision_submission.buy_commitment_amount (is not a valid number)",
		},
		{
			lp: commandspb.SpotLiquidityProvisionSubmission{
				BuyCommitmentAmount:  "10",
				SellCommitmentAmount: "0",
				MarketId:             "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "spot_liquidity_provision_submission.sell_commitment_amount (is not a valid number)",
		},
		{
			lp: commandspb.SpotLiquidityProvisionSubmission{
				BuyCommitmentAmount:  "10",
				SellCommitmentAmount: "-10",
				MarketId:             "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "spot_liquidity_provision_submission.sell_commitment_amount (is not a valid number)",
		},
		{
			lp: commandspb.SpotLiquidityProvisionSubmission{
				BuyCommitmentAmount:  "100",
				SellCommitmentAmount: "100",
				Fee:                  "abcd",
				MarketId:             "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "spot_liquidity_provision_submission.fee (is not a valid value)",
		},

		{
			lp: commandspb.SpotLiquidityProvisionSubmission{
				BuyCommitmentAmount:  "100",
				SellCommitmentAmount: "100",
				Fee:                  "0.1",
			},
			errString: "spot_liquidity_provision_submission.market_id (is required)",
		},
	}

	for i, c := range cases {
		err := commands.CheckSpotLiquidityProvisionSubmission(&c.lp)
		if len(c.errString) <= 0 {
			assert.NoErrorf(t, err, "unexpected error on position: %d", i)
			continue
		}

		assert.Errorf(t, err, "expected error on position: %d", i)
		assert.EqualErrorf(t, err, c.errString, "expected error to match on position: %d", i)
	}
}

func TestCheckSpotLiquidityProvisionCancellation(t *testing.T) {
	type args struct {
		cmd *commandspb.SpotLiquidityProvisionCancellation
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
			errString: "spot_liquidity_provision_cancellation (is required)",
		},
		{
			name: "Should return an error if market_id is not provided",
			args: args{
				cmd: &commandspb.SpotLiquidityProvisionCancellation{
					MarketId: "",
				},
			},
			wantErr:   assert.Error,
			errString: "spot_liquidity_provision_cancellation.market_id (is required)",
		},
		{
			name: "Should succeed if market id is provided",
			args: args{
				cmd: &commandspb.SpotLiquidityProvisionCancellation{
					MarketId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := commands.CheckSpotLiquidityProvisionCancellation(tt.args.cmd)
			tt.wantErr(t, gotErr, fmt.Sprintf("CheckLiquidityProvisionCancellation(%v)", tt.args.cmd))
			if tt.errString != "" {
				assert.EqualError(t, gotErr, tt.errString)
			}
		})
	}
}

func TestCheckSpotLiquidityProvisionAmendment(t *testing.T) {
	type args struct {
		cmd *commandspb.SpotLiquidityProvisionAmendment
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
			errString: "spot_liquidity_provision_amendment (is required)",
		},
		{
			name: "Should return an error when market_id is not provided",
			args: args{
				cmd: &commandspb.SpotLiquidityProvisionAmendment{
					MarketId: "",
				},
			},
			wantErr:   assert.Error,
			errString: "spot_liquidity_provision_amendment.market_id (is required)",
		},
		{
			name: "Should return an error if amendment changes nothing",
			args: args{
				cmd: &commandspb.SpotLiquidityProvisionAmendment{
					MarketId: "abcd",
				},
			},
			wantErr:   assert.Error,
			errString: "spot_liquidity_provision_amendment (is required)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := commands.CheckSpotLiquidityProvisionAmendment(tt.args.cmd)
			tt.wantErr(t, gotErr, fmt.Sprintf("CheckLiquidityProvisionAmendment(%v)", tt.args.cmd))

			if tt.errString != "" {
				assert.EqualError(t, gotErr, tt.errString)
			}
		})
	}
}
