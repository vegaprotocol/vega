package sqlstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/metrics"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type ReferralFeeStats struct {
	*ConnectionSource
}

func NewReferralFeeStats(src *ConnectionSource) *ReferralFeeStats {
	return &ReferralFeeStats{
		ConnectionSource: src,
	}
}

func (rfs *ReferralFeeStats) AddFeeStats(ctx context.Context, stats *entities.ReferralFeeStats) error {
	defer metrics.StartSQLQuery("ReferralFeeStats", "AddFeeStats")()
	_, err := rfs.Connection.Exec(
		ctx,
		`INSERT INTO referral_fee_stats(
                               market_id,
                               asset_id,
                               epoch_seq,
                               total_rewards_paid,
                               referrer_rewards_generated,
                               referees_discount_applied,
                               volume_discount_applied,
                               vega_time
	) values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		stats.MarketID,
		stats.AssetID,
		stats.EpochSeq,
		stats.TotalRewardsPaid,
		stats.ReferrerRewardsGenerated,
		stats.RefereesDiscountApplied,
		stats.VolumeDiscountApplied,
		stats.VegaTime,
	)
	return err
}

func (rfs *ReferralFeeStats) GetFeeStats(ctx context.Context, marketID *entities.MarketID, assetID *entities.AssetID, epochSeq *uint64) (
	*entities.ReferralFeeStats, error,
) {
	defer metrics.StartSQLQuery("ReferralFeeStats", "GetFeeStats")()
	var (
		stats []entities.ReferralFeeStats
		err   error
		args  []interface{}
	)

	if marketID == nil && assetID == nil {
		return nil, errors.New("marketID or assetID must be provided")
	}

	query := `SELECT * FROM referral_fee_stats`
	where := make([]string, 0, 3)

	if epochSeq != nil {
		where = append(where, fmt.Sprintf("epoch_seq = %s", nextBindVar(&args, *epochSeq)))
	}

	if assetID != nil {
		where = append(where, fmt.Sprintf("asset_id = %s", nextBindVar(&args, *assetID)))
	}

	if marketID != nil {
		where = append(where, fmt.Sprintf("market_id = %s", nextBindVar(&args, *marketID)))
	}

	if len(where) > 0 {
		for i, w := range where {
			if i == 0 {
				query = fmt.Sprintf("%s WHERE %s", query, w)
				continue
			}
			query = fmt.Sprintf("%s AND %s", query, w)
		}
	}

	if epochSeq == nil { // we want the most recent stat so order and limit the query
		query = fmt.Sprintf("%s ORDER BY epoch_seq DESC LIMIT 1", query)
	}

	if err = pgxscan.Select(ctx, rfs.Connection, &stats, query, args...); err != nil {
		return nil, err
	}

	if len(stats) == 0 {
		return nil, errors.New("no referral fee stats found")
	}

	return &stats[0], err
}