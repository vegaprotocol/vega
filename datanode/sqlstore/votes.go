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
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type Votes struct {
	*ConnectionSource
}

var votesOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

func NewVotes(connectionSource *ConnectionSource) *Votes {
	d := &Votes{
		ConnectionSource: connectionSource,
	}
	return d
}

func (vs *Votes) Add(ctx context.Context, v entities.Vote) error {
	defer metrics.StartSQLQuery("Votes", "Add")()
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

func (vs *Votes) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Vote, error) {
	defer metrics.StartSQLQuery("Votes", "GetByTxHash")()

	var votes []entities.Vote
	query := `SELECT * FROM votes WHERE tx_hash = $1`
	err := pgxscan.Select(ctx, vs.Connection, &votes, query, txHash)
	if err != nil {
		return nil, fmt.Errorf("querying votes: %w", err)
	}
	return votes, nil
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
