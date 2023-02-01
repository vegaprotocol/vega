package orders

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
)

// RestoreCurrentOrdersSet updates the current order set. As order history is loaded from history segments, after order
// history load, the same order could have multiple marked as the current version (as it was the current version in the
// segment at the time the segment was created), this function ensures only the latest version is marked as current.
func RestoreCurrentOrdersSet(ctx context.Context, conn sqlstore.Connection) error {
	// Note, testing shows that dropping the index on the current flag does not net improve performance of the following update queries

	// Mark all current order versions as false
	_, err := conn.Exec(ctx, "update orders set current = false where current = true")
	if err != nil {
		return fmt.Errorf("failed to execute sql to mark all order versions as not current: %w", err)
	}

	// This query will identify the latest order version for every order and set its current flag to true
	updateCurrentOrders := `with current_orders as (select r.id, max(r.vega_time) as vega_time from orders r group by id),
		current_seq_num as (select o.vega_time, max(o.seq_num) as seq_num from orders o join current_orders co on o.id = co.id and o.vega_time = co.vega_time group by co.id, o.vega_time)
		update orders set current = true from current_seq_num where orders.vega_time = current_seq_num.vega_time and orders.seq_num =  current_seq_num.seq_num`

	_, err = conn.Exec(ctx, updateCurrentOrders)
	if err != nil {
		return fmt.Errorf("failed to execute sql to update current orders: %w", err)
	}

	return nil
}
