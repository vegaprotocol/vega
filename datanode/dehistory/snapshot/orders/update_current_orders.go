package orders

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
)

func UpdateCurrentOrdersState(ctx context.Context, vegaDbConn sqlstore.Connection) error {
	updateVegaTimeToSQL := `WITH vegatimetomapping as (
		SELECT id,
		vega_time,
		COALESCE((LEAD(vega_time) OVER w), 'infinity') as vega_time_to
	FROM orders
	WINDOW w AS (PARTITION BY id order by vega_time))
	UPDATE orders SET vega_time_to=vegatimetomapping.vega_time_to
	FROM vegatimetomapping
	WHERE orders.id=vegatimetomapping.id AND orders.vega_time=vegatimetomapping.vega_time`

	_, err := vegaDbConn.Exec(ctx, updateVegaTimeToSQL)
	if err != nil {
		return fmt.Errorf("failed to execute sql to update vega_time_to: %w", err)
	}

	return nil
}
