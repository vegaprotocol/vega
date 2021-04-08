package steps

import (
	"github.com/cucumber/godog/gherkin"
)

func TheFollowingNetworkParametersAreSet(data *gherkin.DataTable) map[string]interface{} {
	params := map[string]interface{}{}
	for _, row := range TableWrapper(*data).Parse() {
		if minDuration := row.MustDurationSec("market.auction.minimumDuration"); minDuration > 0 {
			params["market.auction.minimumDuration"] = minDuration
			// execsetup.engine.OnMarketAuctionMinimumDurationUpdate(context.Background(), minDuration)
		}
	}
	return params
}
