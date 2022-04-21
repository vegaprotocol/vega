package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Delegations struct {
	*ConnectionSource
}

func NewDelegations(connectionSource *ConnectionSource) *Delegations {
	d := &Delegations{
		ConnectionSource: connectionSource,
	}
	return d
}

func (ds *Delegations) Add(ctx context.Context, d entities.Delegation) error {
	_, err := ds.Connection.Exec(ctx,
		`INSERT INTO delegations(
			party_id,
			node_id,
			epoch_id,
			amount,
			vega_time)
		 VALUES ($1,  $2,  $3,  $4,  $5);`,
		d.PartyID, d.NodeID, d.EpochID, d.Amount, d.VegaTime)
	return err
}

func (ds *Delegations) GetAll(ctx context.Context) ([]entities.Delegation, error) {
	delegations := []entities.Delegation{}
	err := pgxscan.Select(ctx, ds.Connection, &delegations, `
		SELECT * from delegations;`)
	return delegations, err
}

func (ds *Delegations) Get(ctx context.Context,
	partyIDHex *string,
	nodeIDHex *string,
	epochID *int64,
	p *entities.Pagination,
) ([]entities.Delegation, error) {
	query := `SELECT * from delegations`
	args := []interface{}{}

	conditions := []string{}

	if partyIDHex != nil {
		partyID := entities.NewPartyID(*partyIDHex)
		conditions = append(conditions, fmt.Sprintf("party_id=%s", nextBindVar(&args, partyID)))
	}

	if nodeIDHex != nil {
		nodeID := entities.NewNodeID(*nodeIDHex)
		conditions = append(conditions, fmt.Sprintf("node_id=%s", nextBindVar(&args, nodeID)))
	}

	if epochID != nil {
		conditions = append(conditions, fmt.Sprintf("epoch_id=%s", nextBindVar(&args, *epochID)))
	}

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	if p != nil {
		order_cols := []string{"epoch_id", "party_id", "node_id"}
		query, args = orderAndPaginateQuery(query, order_cols, *p, args...)
	}

	delegations := []entities.Delegation{}
	err := pgxscan.Select(ctx, ds.Connection, &delegations, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying delegations: %w", err)
	}
	return delegations, nil
}
