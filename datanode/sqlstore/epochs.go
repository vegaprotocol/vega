// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
)

type Epochs struct {
	*ConnectionSource
}

func NewEpochs(connectionSource *ConnectionSource) *Epochs {
	e := &Epochs{
		ConnectionSource: connectionSource,
	}
	return e
}

func (es *Epochs) Add(ctx context.Context, r entities.Epoch) error {
	defer metrics.StartSQLQuery("Epochs", "Add")()
	_, err := es.Connection.Exec(ctx,
		`INSERT INTO epochs(
			id,
			start_time,
			expire_time,
			end_time,
			tx_hash,
			vega_time)
		 VALUES ($1,  $2,  $3,  $4,  $5, $6)
		 ON CONFLICT (id, vega_time)
		 DO UPDATE SET start_time=EXCLUDED.start_time,
		 	           expire_time=EXCLUDED.expire_time,
		               end_time=EXCLUDED.end_time,
					   tx_hash=EXCLUDED.tx_hash
		 ;`,
		r.ID, r.StartTime, r.ExpireTime, r.EndTime, r.TxHash, r.VegaTime)
	return err
}

func (es *Epochs) GetAll(ctx context.Context) ([]entities.Epoch, error) {
	defer metrics.StartSQLQuery("Epochs", "GetAll")()
	epochs := []entities.Epoch{}
	query := `WITH epochs_current AS (SELECT DISTINCT ON (id) * FROM epochs ORDER BY id, vega_time DESC)
		SELECT e.id, e.start_time, e.expire_time, e.end_time, e.tx_hash, e.vega_time, bs.height first_block, be.height last_block FROM epochs_current AS e
    	LEFT JOIN blocks bs on e.start_time = bs.vega_time
    	LEFT JOIN blocks be on e.end_time = be.vega_time;`
	err := pgxscan.Select(ctx, es.Connection, &epochs, query)
	return epochs, err
}

func (es *Epochs) Get(ctx context.Context, ID int64) (entities.Epoch, error) {
	defer metrics.StartSQLQuery("Epochs", "Get")()
	query := `WITH epochs_current AS (SELECT DISTINCT ON (id) * FROM epochs WHERE id=$1 ORDER BY id, vega_time DESC)
		SELECT e.id, e.start_time, e.expire_time, e.end_time, e.tx_hash, e.vega_time, bs.height first_block, be.height last_block FROM epochs_current AS e
    	LEFT JOIN blocks bs on e.start_time = bs.vega_time
    	LEFT JOIN blocks be on e.end_time = be.vega_time;`

	epoch := entities.Epoch{}
	return epoch, es.wrapE(pgxscan.Get(ctx, es.Connection, &epoch, query, ID))
}

func (es *Epochs) GetCurrent(ctx context.Context) (entities.Epoch, error) {
	query := `WITH epochs_current AS (SELECT DISTINCT ON (id) * FROM epochs ORDER BY id DESC, vega_time DESC)
		SELECT e.id, e.start_time, e.expire_time, e.end_time, e.tx_hash, e.vega_time, bs.height first_block, be.height last_block FROM epochs_current AS e
    	LEFT JOIN blocks bs on e.start_time = bs.vega_time
    	LEFT JOIN blocks be on e.end_time = be.vega_time ORDER BY id DESC, vega_time DESC FETCH FIRST ROW ONLY;`

	epoch := entities.Epoch{}
	defer metrics.StartSQLQuery("Epochs", "GetCurrent")()
	return epoch, es.wrapE(pgxscan.Get(ctx, es.Connection, &epoch, query))
}
