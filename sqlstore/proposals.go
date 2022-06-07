package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/pkg/errors"
)

var ErrProposalNotFound = errors.New("proposal not found")

type Proposals struct {
	*ConnectionSource
}

func NewProposals(connectionSource *ConnectionSource) *Proposals {
	p := &Proposals{
		ConnectionSource: connectionSource,
	}
	return p
}

func (ps *Proposals) Add(ctx context.Context, p entities.Proposal) error {
	defer metrics.StartSQLQuery("Proposals", "Add")()
	_, err := ps.Connection.Exec(ctx,
		`INSERT INTO proposals(
			id,
			reference,
			party_id,
			state,
			terms,
			rationale,
			reason,
			error_details,
			proposal_time,
			vega_time)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6, $7, $8, $9, $10)
		 ON CONFLICT (id, vega_time) DO UPDATE SET
			reference = EXCLUDED.reference,
			party_id = EXCLUDED.party_id,
			state = EXCLUDED.state,
			terms = EXCLUDED.terms,
			rationale = EXCLUDED.rationale,
			reason = EXCLUDED.reason,
			error_details = EXCLUDED.error_details,
			proposal_time = EXCLUDED.proposal_time
			;
		 `,
		p.ID, p.Reference, p.PartyID, p.State, p.Terms, p.Rationale, p.Reason, p.ErrorDetails, p.ProposalTime, p.VegaTime)
	return err
}

func (ps *Proposals) GetByID(ctx context.Context, id string) (entities.Proposal, error) {
	defer metrics.StartSQLQuery("Proposals", "GetByID")()
	var p entities.Proposal
	query := `SELECT * FROM proposals_current WHERE id=$1`
	err := pgxscan.Get(ctx, ps.Connection, &p, query, entities.NewProposalID(id))
	if pgxscan.NotFound(err) {
		return p, fmt.Errorf("'%v': %w", id, ErrProposalNotFound)
	}

	return p, err
}

func (ps *Proposals) GetByReference(ctx context.Context, ref string) (entities.Proposal, error) {
	defer metrics.StartSQLQuery("Proposals", "GetByReference")()
	var p entities.Proposal
	query := `SELECT * FROM proposals_current WHERE reference=$1 LIMIT 1`
	err := pgxscan.Get(ctx, ps.Connection, &p, query, ref)
	if pgxscan.NotFound(err) {
		return p, fmt.Errorf("'%v': %w", ref, ErrProposalNotFound)
	}

	return p, err
}

func (ps *Proposals) Get(ctx context.Context,
	inState *entities.ProposalState,
	partyIDStr *string,
	proposalType *entities.ProposalType,
) ([]entities.Proposal, error) {
	query := `SELECT * FROM proposals_current`
	args := []interface{}{}

	conditions := []string{}

	if inState != nil {
		conditions = append(conditions, fmt.Sprintf("state=%s", nextBindVar(&args, *inState)))
	}

	if partyIDStr != nil {
		partyID := entities.NewPartyID(*partyIDStr)
		conditions = append(conditions, fmt.Sprintf("party_id=%s", nextBindVar(&args, partyID)))
	}

	if proposalType != nil {
		conditions = append(conditions, fmt.Sprintf("terms ? %s", nextBindVar(&args, *proposalType)))
	}

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	proposals := []entities.Proposal{}
	defer metrics.StartSQLQuery("Proposals", "Get")()
	err := pgxscan.Select(ctx, ps.Connection, &proposals, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying proposals: %w", err)
	}
	return proposals, nil

}
