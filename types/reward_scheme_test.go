package types_test

import (
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

func TestGetString(t *testing.T) {
	param := &types.RewardSchemeParam{
		Name:  "param1",
		Type:  "string",
		Value: "value1",
	}
	require.Equal(t, "value1", param.GetString())
}

func TestGetFloatFailure(t *testing.T) {
	param := &types.RewardSchemeParam{
		Name:  "param1",
		Type:  "float",
		Value: "not a number",
	}
	value, err := param.GetFloat()
	require.Error(t, errors.New("mismatch between requested type and configured type for param1"), err)
	require.Equal(t, 0.0, value)
}

func TestGetFloatSuccess(t *testing.T) {
	param := &types.RewardSchemeParam{
		Name:  "param1",
		Type:  "float",
		Value: "123.456",
	}
	value, err := param.GetFloat()
	require.NoError(t, err)
	require.Equal(t, 123.456, value)
}

func TestGetUintFailure(t *testing.T) {
	param := &types.RewardSchemeParam{
		Name:  "param1",
		Type:  "uint",
		Value: "not a number",
	}
	value, err := param.GetUint()
	require.Error(t, errors.New("mismatch between requested type and configured type for param1"), err)
	require.Nil(t, value)
}

func TestGetUintZero(t *testing.T) {
	param := &types.RewardSchemeParam{
		Name:  "param1",
		Type:  "uint",
		Value: "0",
	}
	value, err := param.GetUint()
	require.NoError(t, err)
	require.Equal(t, num.Zero(), value)
}

func TestGetUintSuccess(t *testing.T) {
	param := &types.RewardSchemeParam{
		Name:  "param1",
		Type:  "uint",
		Value: "100",
	}
	value, err := param.GetUint()
	require.NoError(t, err)
	require.Equal(t, num.NewUint(100), value)
}

func TestGetUintError(t *testing.T) {
	param := &types.RewardSchemeParam{
		Name:  "param1",
		Type:  "uint",
		Value: "100.1",
	}
	value, err := param.GetUint()
	require.Nil(t, value)
	require.Error(t, errors.New("mismatch between requested type and configured type for param1"), err)
}

func TestIsActiveBeforeStart(t *testing.T) {
	now := time.Now()

	rs := &types.RewardScheme{
		StartTime: now.Add(time.Second * 10),
	}

	require.Equal(t, false, rs.IsActive(now))
}

func TestIsActiveNoEnd(t *testing.T) {
	now := time.Now()

	rs := &types.RewardScheme{
		StartTime: now.Add(-10 * time.Second),
	}

	require.Equal(t, true, rs.IsActive(now))
}

func TestIsActiveBeforeEnd(t *testing.T) {
	now := time.Now()
	endTime := now.Add(10 * time.Second)
	rs := &types.RewardScheme{
		StartTime: now.Add(-10 * time.Second),
		EndTime:   &endTime,
	}

	require.Equal(t, true, rs.IsActive(now))
}

func TestIsActiveAfterEnd(t *testing.T) {
	now := time.Now()
	endTime := now.Add(10 * time.Second)
	rs := &types.RewardScheme{
		StartTime: now.Add(-10 * time.Second),
		EndTime:   &endTime,
	}

	require.Equal(t, false, rs.IsActive(endTime.Add(1*time.Second)))
}

func TestGetRewardBalanceIsZero(t *testing.T) {
	rs := &types.RewardScheme{}
	epoch := types.Epoch{}
	rewardForScheme, err := rs.GetReward(num.Zero(), epoch)
	require.Nil(t, err)
	require.Equal(t, num.Zero(), rewardForScheme)
}

func TestGetRewardFractional(t *testing.T) {
	rs := &types.RewardScheme{
		PayoutType:     types.PayoutFractional,
		PayoutFraction: num.DecimalFromFloat(0.5),
	}
	epoch := types.Epoch{}
	rewardForScheme, err := rs.GetReward(num.NewUint(10000), epoch)
	require.Nil(t, err)
	require.Equal(t, num.NewUint(5000), rewardForScheme)
}

func TestGetRewardBalancedNoEndTime(t *testing.T) {
	rs := &types.RewardScheme{
		PayoutType: types.PayoutBalanced,
	}
	epoch := types.Epoch{}
	rewardForScheme, err := rs.GetReward(num.NewUint(10000), epoch)
	require.Error(t, types.ErrRewardSchemeMisconfiguration, err)
	require.Nil(t, rewardForScheme)
}

func TestGetRewardBalanced(t *testing.T) {
	now := time.Now()
	rewardEndTime := now.Add(100 * time.Second)
	rs := &types.RewardScheme{
		PayoutType: types.PayoutBalanced,
		EndTime:    &rewardEndTime,
	}

	// epoch is 10 seconds
	epoch := types.Epoch{
		StartTime: now.Add(-10 * time.Second),
		EndTime:   now,
	}
	rewardForScheme, err := rs.GetReward(num.NewUint(10000), epoch)
	require.Nil(t, err)
	require.Equal(t, num.NewUint(1000), rewardForScheme)
}
