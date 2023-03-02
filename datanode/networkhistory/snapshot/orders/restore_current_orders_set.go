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

	query := `WITH current_orders AS (
  SELECT id, MAX(vega_time) AS max_vega_time
  FROM orders
  WHERE current = true
  GROUP BY id
)
UPDATE orders o
    SET current = CASE
        WHEN o.vega_time = co.max_vega_time AND o.seq_num = (
            SELECT MAX(seq_num)
            FROM orders
            WHERE id = o.id AND vega_time = co.max_vega_time
        ) THEN true ELSE false
    END
FROM current_orders co
WHERE o.id = co.id AND o.current = true;
`
	// Mark all current order versions as false
	if _, err := conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to execute query to mark all but the most recent row as current order: %+w", err)
	}
	return nil
}
