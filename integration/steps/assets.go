package steps

import (
	"code.vegaprotocol.io/vega/integration/stubs"

	"github.com/cucumber/godog"
)

func RegisterAsset(tbl *godog.Table, asset *stubs.AssetStub) error {
	rows := StrictParseTable(tbl, []string{
		"id",
		"decimal places",
	}, nil)
	for _, row := range rows {
		asset.Register(
			row.MustStr("id"),
			row.MustU64("decimal places"),
		)
	}
	return nil
}
