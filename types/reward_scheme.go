package types

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/types/num"
)

// RewardSchemeScope defines the scope of the reward scheme.
type RewardSchemeScope int

const (
	RewardSchemeScopeUndefined RewardSchemeScope = iota
	RewardSchemeScopeNetwork
	RewardSchemeScopeAsset
	RewardSchemeScopeMarket
)

// RewardSchemeType defines the type of the reward scheme - currently only staking and delegation is supported.
type RewardSchemeType int

const (
	RewardSchemeUndefined RewardSchemeType = iota
	RewardSchemeStakingAndDelegation
)

// PayoutType - fractional or balanced.
type PayoutType int

const (
	PayoutUndefined PayoutType = iota
	PayoutFractional
	PayoutBalanced
)

// RewardSchemeParam defines parameters for the reward scheme.
type RewardSchemeParam struct {
	Name  string
	Type  string
	Value string
}

// RewardScheme defines an instance of a reward strategy and its parameters and scope.
type RewardScheme struct {
	SchemeID                  string
	Type                      RewardSchemeType
	ScopeType                 RewardSchemeScope
	Scope                     string
	Parameters                map[string]RewardSchemeParam
	StartTime                 time.Time
	EndTime                   *time.Time
	PayoutType                PayoutType
	PayoutFraction            float64
	MaxPayoutPerAssetPerParty map[string]*num.Uint
	PayoutDelay               time.Duration
	RewardPoolAccountIDs      []string
}

func (rsp RewardSchemeParam) GetString() string {
	return rsp.Value
}

func (rsp RewardSchemeParam) GetFloat() (float64, error) {
	if rsp.Type == "float" {
		num, err := num.DecimalFromString(rsp.Value)
		if err != nil {
			return 0, err
		}
		res, _ := num.Float64()
		return res, nil
	}
	return 0, errors.New("mismatch between requested type and configured type for" + rsp.Name)
}

func (rsp RewardSchemeParam) GetUint() (*num.Uint, error) {
	if rsp.Type == "uint" {
		res, err := num.UintFromString(rsp.Value, 10)
		if err {
			return nil, errors.New("mismatch between requested type and configured type " + rsp.Name)
		}
		return res, nil
	}
	return nil, errors.New("mismatch between requested type and configured type " + rsp.Name)
}

func (rsp RewardSchemeParam) GetDecimal() (*num.Decimal, error) {
	if rsp.Type == "float" {
		res, err := num.DecimalFromString(rsp.Value)
		if err != nil {
			return nil, errors.New("mismatch between requested type and configured type " + rsp.Name)
		}
		return &res, nil
	}
	return nil, errors.New("mismatch between requested type and configured type " + rsp.Name)
}

// ErrRewardSchemeMisconfiguration is returned when trying to calculate the reward for a given account balance and the scheme has incompatible end time and payout type
// this should never happen as the reward scheme needs to be validated prior to being added but just to be safe.
var ErrRewardSchemeMisconfiguration = errors.New("payout type balanced is incompatible with having no end time")

// IsActive returns true if the current time is after the scheme start time and not after the scheme end time.
func (rs *RewardScheme) IsActive(now time.Time) bool {
	return !now.Before(rs.StartTime) && (rs.EndTime == nil || !now.After(*rs.EndTime))
}

// GetReward calculates the reward given the pool balance and the reward scheme parameters.
func (rs *RewardScheme) GetReward(rewardPoolBalance *num.Uint, epoch Epoch) (*num.Uint, error) {
	if rewardPoolBalance.IsZero() {
		return num.Zero(), nil
	}

	var rewardBalance *num.Uint
	if rs.PayoutType == PayoutFractional {
		rewardBalance, _ = num.UintFromDecimal(num.NewDecimalFromFloat(rs.PayoutFraction).Mul(rewardPoolBalance.ToDecimal()))
	} else {
		if rs.EndTime == nil {
			return nil, ErrRewardSchemeMisconfiguration
		}
		epochLength := epoch.EndTime.UnixNano() - epoch.StartTime.UnixNano()
		numberOfEpochsTillExpiry := (rs.EndTime.UnixNano() - epoch.EndTime.UnixNano()) / epochLength
		rewardBalance = num.Zero().Div(rewardPoolBalance, num.NewUint(uint64(numberOfEpochsTillExpiry)))
	}
	return rewardBalance, nil
}
