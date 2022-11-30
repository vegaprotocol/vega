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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type Votes struct {
	*ConnectionSource
}

var (
	incClauses = map[entities.VoteValue]string{
		entities.VoteValueNo:  "no_votes = no_votes + 1",
		entities.VoteValueYes: "yes_votes = yes_votes + 1",
	}

	decClauses = map[entities.VoteValue]string{
		entities.VoteValueNo:  "no_votes = no_votes - 1",
		entities.VoteValueYes: "yes_votes = yes_votes - 1",
	}
)

var votesOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

func NewVotes(connectionSource *ConnectionSource) *Votes {
	d := &Votes{
		ConnectionSource: connectionSource,
	}
	return d
}

func (vs *Votes) addTotal(ctx context.Context, v entities.Vote) error {
	query := `SELECT * FROM votes_current WHERE proposal_id = $1 AND party_id = $2`
	votes := []entities.Vote{}
	err := pgxscan.Select(ctx, vs.Connection, &votes, query, v.ProposalID, v.PartyID)
	if err != nil || len(votes) == 0 {
		// no previous vote, just increment the correct total
		query = fmt.Sprintf(`UPDATE proposals SET %s WHERE id = $1`, incClauses[v.Value])
	} else if votes[0].Value == v.Value {
		// we had a previous vote registered, but it had the same value - totals should be fine
		return nil
	} else {
		// Vote value changed, decrement the previous value total, and increment the other
		clauses := make([]string, 0, 2)
		clauses = append(clauses, incClauses[v.Value])
		clauses = append(clauses, decClauses[votes[0].Value])
		query = fmt.Sprintf(`UPDATE proposals SET %s WHERE proposal_id = $1`, strings.Join(clauses, ", "))
	}
	// update the totals:
	_, err = vs.Connection.Exec(ctx, query, v.ProposalID)
	return err
}

func (vs *Votes) Add(ctx context.Context, v entities.Vote) error {
	defer metrics.StartSQLQuery("Votes", "Add")()
	// this is a bit clunky but if we could not set the totals properly, we probably are dealing with a vote
	// for a proposal that doesn't exist yet, or something is seriously wrong
	if err := vs.addTotal(ctx, c); err != nil {
		return err
	}
	_, err := vs.Connection.Exec(ctx,
		`INSERT INTO votes(
			proposal_id,
			party_id,
			value,
			tx_hash,
			vega_time,
			initial_time,
			total_governance_token_balance,
			total_governance_token_weight,
			total_equity_like_share_weight
		)
		 VALUES ($1,  $2,  $3,  $4,  $5, $6, $7, $8, $9)
		 ON CONFLICT (proposal_id, party_id, vega_time) DO UPDATE SET
			value = EXCLUDED.value,
			total_governance_token_balance =EXCLUDED.total_governance_token_balance,
			total_governance_token_weight = EXCLUDED.total_governance_token_weight,
			total_equity_like_share_weight = EXCLUDED.total_equity_like_share_weight,
			tx_hash = EXCLUDED.tx_hash;
		`,
		v.ProposalID, v.PartyID, v.Value, v.TxHash, v.VegaTime, v.InitialTime,
		v.TotalGovernanceTokenBalance, v.TotalGovernanceTokenWeight, v.TotalEquityLikeShareWeight)
	return err
}

func (vs *Votes) GetYesVotesForProposal(ctx context.Context, proposalIDStr string) ([]entities.Vote, error) {
	defer metrics.StartSQLQuery("Votes", "GetYesVotesForProposal")()
	yes := entities.VoteValueYes
	return vs.Get(ctx, &proposalIDStr, nil, &yes)
}

func (vs *Votes) GetNoVotesForProposal(ctx context.Context, proposalIDStr string) ([]entities.Vote, error) {
	defer metrics.StartSQLQuery("Votes", "GetNoVotesForProposal")()
	no := entities.VoteValueNo
	return vs.Get(ctx, &proposalIDStr, nil, &no)
}

func (vs *Votes) GetByParty(ctx context.Context, partyIDStr string) ([]entities.Vote, error) {
	defer metrics.StartSQLQuery("Votes", "GetByParty")()
	return vs.Get(ctx, nil, &partyIDStr, nil)
}

func (vs *Votes) GetByPartyConnection(ctx context.Context, partyIDStr string, pagination entities.CursorPagination) ([]entities.Vote, entities.PageInfo, error) {
	args := make([]interface{}, 0)
	query := fmt.Sprintf(`select * from votes_current where party_id=%s`, nextBindVar(&args, entities.PartyID(partyIDStr)))

	var (
		votes    []entities.Vote
		pageInfo entities.PageInfo
		err      error
	)

	query, args, err = PaginateQuery[entities.VoteCursor](query, args, votesOrdering, pagination)
	if err != nil {
		return votes, pageInfo, err
	}

	if err = pgxscan.Select(ctx, vs.Connection, &votes, query, args...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	votes, pageInfo = entities.PageEntities[*v2.VoteEdge](votes, pagination)
	return votes, pageInfo, nil
}

func (vs *Votes) GetConnection(
	ctx context.Context,
	proposalIDStr, partyIDStr *string,
	pagination entities.CursorPagination,
) ([]entities.Vote, entities.PageInfo, error) {
	query := `SELECT * FROM votes_current`
	args := []interface{}{}

	conditions := []string{}

	if proposalIDStr != nil {
		proposalID := entities.ProposalID(*proposalIDStr)
		conditions = append(conditions, fmt.Sprintf("proposal_id=%s", nextBindVar(&args, proposalID)))
	}

	if partyIDStr != nil {
		partyID := entities.PartyID(*partyIDStr)
		conditions = append(conditions, fmt.Sprintf("party_id=%s", nextBindVar(&args, partyID)))
	}

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	var (
		votes    []entities.Vote
		pageInfo entities.PageInfo
		err      error
	)

	query, args, err = PaginateQuery[entities.VoteCursor](query, args, votesOrdering, pagination)
	if err != nil {
		return votes, pageInfo, err
	}

	if err = pgxscan.Select(ctx, vs.Connection, &votes, query, args...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	votes, pageInfo = entities.PageEntities[*v2.VoteEdge](votes, pagination)
	return votes, pageInfo, nil
}

func (vs *Votes) Get(ctx context.Context,
	proposalIDStr *string,
	partyIDStr *string,
	value *entities.VoteValue,
) ([]entities.Vote, error) {
	query := `SELECT * FROM votes_current`
	args := []interface{}{}

	conditions := []string{}

	if proposalIDStr != nil {
		proposalID := entities.ProposalID(*proposalIDStr)
		conditions = append(conditions, fmt.Sprintf("proposal_id=%s", nextBindVar(&args, proposalID)))
	}

	if partyIDStr != nil {
		partyID := entities.PartyID(*partyIDStr)
		conditions = append(conditions, fmt.Sprintf("party_id=%s", nextBindVar(&args, partyID)))
	}

	if value != nil {
		conditions = append(conditions, fmt.Sprintf("value=%s", nextBindVar(&args, *value)))
	}

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	votes := []entities.Vote{}
	err := pgxscan.Select(ctx, vs.Connection, &votes, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying votes: %w", err)
	}
	return votes, nil
}
