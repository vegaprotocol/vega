// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"fmt"
	"strings"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
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

func getOpenStateProposalsQuery(inState *entities.ProposalState, conditions []string, pagination entities.CursorPagination,
	sorting Sorting, cmp Compare, pc *entities.ProposalCursor, pageForward bool, args ...interface{}) (string, []interface{}) {
	// if we're querying for a specific state and it's not the Open state,
	// or if we are paging forward and the current state is not the open state
	// then we do not need to query for any open state proposals
	if (inState != nil && *inState != entities.ProposalStateOpen) ||
		(pageForward && pc.State != entities.ProposalStateUnspecified && pc.State != entities.ProposalStateOpen) {
		// we aren't interested in open proposals so the query should be empty
		return "", args
	}

	conditions = append([]string{
		fmt.Sprintf("state=%s", nextBindVar(&args, entities.ProposalStateOpen)),
	}, conditions...)

	query := `select * from proposals_current`

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	cursorParams := make([]CursorQueryParameter, 0)

	// only add the vega_time constraint if the cursor is pointing to a record whose state is open
	if pc.State == entities.ProposalStateOpen {
		cursorParams = append(cursorParams,
			NewCursorQueryParameter("vega_time", sorting, cmp, pc.VegaTime),
			NewCursorQueryParameter("id", sorting, cmp, entities.NewProposalID(pc.ID)),
		)
	} else {
		cursorParams = append(cursorParams,
			NewCursorQueryParameter("vega_time", sorting, cmp, nil),
			NewCursorQueryParameter("id", sorting, cmp, nil),
		)
	}

	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	return query, args
}

func getOtherStateProposalsQuery(inState *entities.ProposalState, conditions []string, pagination entities.CursorPagination,
	sorting Sorting, cmp Compare, pc *entities.ProposalCursor, pageForward bool, args ...interface{}) (string, []interface{}) {
	// if we're filtering for state and the state is open,
	// or we're paging forward, and the cursor has reached the open proposals
	// then we don't need to return any non-open proposal results
	if (inState != nil && *inState == entities.ProposalStateOpen) || (!pageForward && pc.State == entities.ProposalStateOpen) {
		// the open state query should already be providing the correct query for this
		return "", args
	}

	if inState == nil {
		conditions = append([]string{
			fmt.Sprintf("state!=%s", nextBindVar(&args, entities.ProposalStateOpen)),
		}, conditions...)
	} else {
		conditions = append([]string{
			fmt.Sprintf("state=%s", nextBindVar(&args, *inState)),
		}, conditions...)
	}
	query := `select * from proposals_current`

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	cursorParams := make([]CursorQueryParameter, 0)

	if pc.State == entities.ProposalStateOpen {
		// we want all the current proposals that are not open, ordered by vega_time
		cursorParams = append(cursorParams,
			NewCursorQueryParameter("vega_time", sorting, "", nil),
			NewCursorQueryParameter("id", sorting, "", nil),
		)
	} else {
		cursorParams = append(cursorParams,
			NewCursorQueryParameter("vega_time", sorting, cmp, pc.VegaTime),
			NewCursorQueryParameter("id", sorting, cmp, nil),
		)
	}

	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)
	return query, args
}

func (ps *Proposals) Get(ctx context.Context,
	inState *entities.ProposalState,
	partyIDStr *string,
	proposalType *entities.ProposalType,
	pagination entities.CursorPagination,
) ([]entities.Proposal, entities.PageInfo, error) {

	// This one is a bit tricky because we want all the open proposals listed at the top, sorted by date
	// then other proposals in date order regardless of state.

	// In order to do this, we need to construct a union of proposals where state = open, order by vega_time
	// and state != open, order by vega_time
	// If the cursor has been set, and we're traversing forward (newest-oldest), then we need to check if the
	// state of the cursor is = open. If it is then we should append the open state proposals with the non-open state
	// proposals.
	// If the cursor state is != open, we have navigated passed the open state proposals and only need the non-open state proposals.

	// If the cursor has been set and we're traversing backward (newest-oldest), then we need to check if the
	// state of the cursor is = open. If it is then we should only return the proposals where state = open as we've already navigated
	// passed all the non-open proposals.
	// if the state of the cursor is != open, then we need to append all the proposals where the state = open after the proposals where
	// state != open.

	// This combined results of both queries is then wrapped with another select which should return the appropriate number of rows that
	// are required for the pagination to determine whether or not there are any next/previous rows for the pageInfo.
	var (
		pageInfo        entities.PageInfo
		stateOpenQuery  string
		stateOtherQuery string
		stateOpenArgs   []interface{}
		stateOtherArgs  []interface{}
	)
	args := make([]interface{}, 0)
	sorting, cmp, cursor := extractPaginationInfo(pagination)

	pc := &entities.ProposalCursor{}

	if cursor != "" {
		err := pc.Parse(cursor)
		if err != nil {
			return nil, pageInfo, err
		}
	}

	pageForward := pagination.HasForward() || (!pagination.HasForward() && !pagination.HasBackward())
	var conditions []string

	if partyIDStr != nil {
		partyID := entities.NewPartyID(*partyIDStr)
		conditions = append(conditions, fmt.Sprintf("party_id=%s", nextBindVar(&args, partyID)))
	}

	if proposalType != nil {
		conditions = append(conditions, fmt.Sprintf("terms ? %s", nextBindVar(&args, *proposalType)))
	}

	stateOpenQuery, stateOpenArgs = getOpenStateProposalsQuery(inState, conditions, pagination, sorting, cmp, pc, pageForward, args...)
	stateOtherQuery, stateOtherArgs = getOtherStateProposalsQuery(inState, conditions, pagination, sorting, cmp, pc, pageForward, args...)

	batch := &pgx.Batch{}

	if stateOpenQuery != "" {
		batch.Queue(stateOpenQuery, stateOpenArgs...)
	}

	if stateOtherQuery != "" {
		batch.Queue(stateOtherQuery, stateOtherArgs...)
	}

	defer metrics.StartSQLQuery("Proposals", "Get")()
	results := ps.Connection.SendBatch(ctx, batch)
	defer results.Close()

	proposals := make([]entities.Proposal, 0)

	for {
		rows, err := results.Query()
		if err != nil {
			break
		}

		var props []entities.Proposal

		if err := pgxscan.ScanAll(&props, rows); err != nil {
			return nil, pageInfo, fmt.Errorf("querying proposals: %w", err)
		}

		if pageForward {
			proposals = append(proposals, props...)
		} else {
			proposals = append(props, proposals...)
		}
	}

	if limit := calculateLimit(pagination); limit > 0 && limit < len(proposals) {
		proposals = proposals[:limit]
	}

	proposals, pageInfo = entities.PageEntities[*v2.GovernanceDataEdge](proposals, pagination)
	return proposals, pageInfo, nil
}
