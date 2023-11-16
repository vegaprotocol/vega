// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/georgysavva/scany/pgxscan"
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

func (es *Epochs) Get(ctx context.Context, ID uint64) (entities.Epoch, error) {
	defer metrics.StartSQLQuery("Epochs", "Get")()
	query := `WITH epochs_current AS (SELECT DISTINCT ON (id) * FROM epochs WHERE id=$1 ORDER BY id, vega_time DESC)
		SELECT e.id, e.start_time, e.expire_time, e.end_time, e.tx_hash, e.vega_time, bs.height first_block, be.height last_block FROM epochs_current AS e
    	LEFT JOIN blocks bs on e.start_time = bs.vega_time
    	LEFT JOIN blocks be on e.end_time = be.vega_time;`

	epoch := entities.Epoch{}
	return epoch, es.wrapE(pgxscan.Get(ctx, es.Connection, &epoch, query, ID))
}

func (es *Epochs) GetByBlock(ctx context.Context, height uint64) (entities.Epoch, error) {
	defer metrics.StartSQLQuery("Epochs", "GetByBlock")()
	query := `WITH epochs_current AS (SELECT DISTINCT ON (id) * FROM epochs ORDER BY id, vega_time DESC)
		SELECT e.id, e.start_time, e.expire_time, e.end_time, e.tx_hash, e.vega_time, bs.height first_block, be.height last_block FROM epochs_current AS e
		LEFT JOIN blocks bs on e.start_time = bs.vega_time
		LEFT JOIN blocks be on e.end_time = be.vega_time WHERE bs.height <= $1 AND be.height > $1;`

	epoch := entities.Epoch{}
	return epoch, es.wrapE(pgxscan.Get(ctx, es.Connection, &epoch, query, height))
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
