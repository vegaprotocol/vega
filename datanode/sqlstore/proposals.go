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
	"fmt"
	"strings"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type Proposals struct {
	*ConnectionSource
}

var proposalsOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: ASC},
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
			vega_time,
			required_majority,
			required_participation,
			required_lp_majority,
			required_lp_participation,
			tx_hash)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 ON CONFLICT (id, vega_time) DO UPDATE SET
			reference = EXCLUDED.reference,
			party_id = EXCLUDED.party_id,
			state = EXCLUDED.state,
			terms = EXCLUDED.terms,
			rationale = EXCLUDED.rationale,
			reason = EXCLUDED.reason,
			error_details = EXCLUDED.error_details,
			proposal_time = EXCLUDED.proposal_time,
			tx_hash = EXCLUDED.tx_hash
			;
		 `,
		p.ID, p.Reference, p.PartyID, p.State, p.Terms, p.Rationale, p.Reason, p.ErrorDetails, p.ProposalTime, p.VegaTime, p.RequiredMajority, p.RequiredParticipation, p.RequiredLPMajority, p.RequiredLPParticipation, p.TxHash)
	return err
}

func (ps *Proposals) GetByID(ctx context.Context, id string) (entities.Proposal, error) {
	defer metrics.StartSQLQuery("Proposals", "GetByID")()
	var p entities.Proposal
	query := `SELECT * FROM proposals_current WHERE id=$1`

	return p, ps.wrapE(pgxscan.Get(ctx, ps.Connection, &p, query, entities.ProposalID(id)))
}

func (ps *Proposals) GetByReference(ctx context.Context, ref string) (entities.Proposal, error) {
	defer metrics.StartSQLQuery("Proposals", "GetByReference")()
	var p entities.Proposal
	query := `SELECT * FROM proposals_current WHERE reference=$1 LIMIT 1`
	return p, ps.wrapE(pgxscan.Get(ctx, ps.Connection, &p, query, ref))
}

func (ps *Proposals) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Proposal, error) {
	defer metrics.StartSQLQuery("Proposals", "GetByTxHash")()

	var proposals []entities.Proposal
	query := `SELECT * FROM proposals WHERE tx_hash=$1`
	err := pgxscan.Select(ctx, ps.Connection, &proposals, query, txHash)
	if err != nil {
		return nil, ps.wrapE(err)
	}

	return proposals, nil
}

func getOpenStateProposalsQuery(inState *entities.ProposalState, conditions []string, pagination entities.CursorPagination,
	pc *entities.ProposalCursor, pageForward bool, args ...interface{},
) (string, []interface{}, error) {
	// if we're querying for a specific state and it's not the Open state,
	// or if we are paging forward and the current state is not the open state
	// then we do not need to query for any open state proposals
	if (inState != nil && *inState != entities.ProposalStateOpen) ||
		(pageForward && pc.State != entities.ProposalStateUnspecified && pc.State != entities.ProposalStateOpen) {
		// we aren't interested in open proposals so the query should be empty
		return "", args, nil
	}

	if pc.State != entities.ProposalStateOpen {
		if pagination.HasForward() {
			pagination.Forward.Cursor = nil
		} else if pagination.HasBackward() {
			pagination.Backward.Cursor = nil
		}
	}

	conditions = append([]string{
		fmt.Sprintf("state=%s", nextBindVar(&args, entities.ProposalStateOpen)),
	}, conditions...)

	query := `select * from proposals_current`

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	var err error
	query, args, err = PaginateQuery[entities.ProposalCursor](query, args, proposalsOrdering, pagination)
	if err != nil {
		return "", args, err
	}

	return query, args, nil
}

func getOtherStateProposalsQuery(inState *entities.ProposalState, conditions []string, pagination entities.CursorPagination,
	pc *entities.ProposalCursor, pageForward bool, args ...interface{},
) (string, []interface{}, error) {
	// if we're filtering for state and the state is open,
	// or we're paging forward, and the cursor has reached the open proposals
	// then we don't need to return any non-open proposal results
	if (inState != nil && *inState == entities.ProposalStateOpen) || (!pageForward && pc.State == entities.ProposalStateOpen) {
		// the open state query should already be providing the correct query for this
		return "", args, nil
	}

	if pagination.HasForward() {
		if pc.State == entities.ProposalStateOpen || pc.State == entities.ProposalStateUnspecified {
			pagination.Forward.Cursor = nil
		}
	} else if pagination.HasBackward() {
		if pc.State == entities.ProposalStateOpen || pc.State == entities.ProposalStateUnspecified {
			pagination.Backward.Cursor = nil
		}
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

	var err error
	query, args, err = PaginateQuery[entities.ProposalCursor](query, args, proposalsOrdering, pagination)
	if err != nil {
		return "", args, err
	}
	return query, args, nil
}

func clonePagination(p entities.CursorPagination) (entities.CursorPagination, error) {
	var first, last int32
	var after, before string

	var pFirst, pLast *int32
	var pAfter, pBefore *string

	if p.HasForward() {
		first = *p.Forward.Limit
		pFirst = &first
		if p.Forward.HasCursor() {
			after = p.Forward.Cursor.Encode()
			pAfter = &after
		}
	}

	if p.HasBackward() {
		last = *p.Backward.Limit
		pLast = &last
		if p.Backward.HasCursor() {
			before = p.Backward.Cursor.Encode()
			pBefore = &before
		}
	}

	return entities.NewCursorPagination(pFirst, pAfter, pLast, pBefore, p.NewestFirst)
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
	cursor := extractCursorFromPagination(pagination)

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
		partyID := entities.PartyID(*partyIDStr)
		conditions = append(conditions, fmt.Sprintf("party_id=%s", nextBindVar(&args, partyID)))
	}

	if proposalType != nil {
		conditions = append(conditions, fmt.Sprintf("terms ? %s", nextBindVar(&args, proposalType.String())))
	}

	var err error
	var openPagination, otherPagination entities.CursorPagination
	// we need to clone the pagination objects because we need to alter the pagination data for the different states
	// to support the required ordering of the data
	openPagination, err = clonePagination(pagination)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("invalid pagination: %w", err)
	}
	otherPagination, err = clonePagination(pagination)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("invalid pagination: %w", err)
	}

	stateOpenQuery, stateOpenArgs, err = getOpenStateProposalsQuery(inState, conditions, openPagination, pc, pageForward, args...)
	if err != nil {
		return nil, pageInfo, err
	}
	stateOtherQuery, stateOtherArgs, err = getOtherStateProposalsQuery(inState, conditions, otherPagination, pc, pageForward, args...)
	if err != nil {
		return nil, pageInfo, err
	}

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
