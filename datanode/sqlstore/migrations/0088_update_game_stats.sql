-- +goose Up

create or replace view game_stats as
WITH
  game_epochs AS (
    SELECT DISTINCT game_id, epoch_id
    FROM rewards
    WHERE game_id IS NOT NULL
  ),
  dispatch_strategies AS (
    SELECT game_id, dispatch_strategy FROM transfers WHERE transfer_type = 'Recurring' ORDER BY vega_time DESC LIMIT 1
  ),
  game_rewards AS (
    SELECT r.*,
           t.dispatch_strategy,
           tm.team_id,
           tmr.rank AS member_rank,
           tr.rank AS team_rank,
           tmr.total_rewards,
           tr.total_rewards AS team_total_rewards,
           'ENTITY_SCOPE_TEAMS' AS entity_scope
    FROM rewards r
      JOIN game_epochs ge ON r.game_id = ge.game_id AND r.epoch_id = ge.epoch_id
      JOIN dispatch_strategies t ON r.game_id = t.game_id AND t.dispatch_strategy ->> 'entity_scope' = '2'
      JOIN team_members tm ON r.party_id = tm.party_id
      LEFT JOIN game_team_rankings tr ON r.game_id = tr.game_id AND r.epoch_id = tr.epoch_id AND tm.team_id = tr.team_id
      LEFT JOIN game_team_member_rankings tmr
                ON r.game_id = tmr.game_id AND r.epoch_id = tmr.epoch_id AND tm.team_id = tmr.team_id AND
                   r.party_id = tmr.party_id
    UNION ALL
    SELECT r.*,
           t.dispatch_strategy,
           NULL AS team_id,
           tmr.rank AS member_rank,
           NULL AS team_rank,
           tmr.total_rewards,
           NULL AS team_total_rewards,
           'ENTITY_SCOPE_INDIVIDUALS' AS entity_scope
    FROM rewards r
      JOIN game_epochs ge ON r.game_id = ge.game_id AND r.epoch_id = ge.epoch_id
      JOIN dispatch_strategies t ON r.game_id = t.game_id AND dispatch_strategy ->> 'entity_scope' = '1'
      LEFT JOIN game_individual_rankings tmr
                ON r.game_id = tmr.game_id AND r.epoch_id = tmr.epoch_id AND r.party_id = tmr.party_id
  )
SELECT s.*
FROM game_rewards s;

create or replace view game_stats_current as
with game_epochs as (
    select game_id, max(epoch_id) as epoch_id
    from rewards
    where game_id is not null
    group by game_id
),
dispatch_strategies AS (
    SELECT game_id, dispatch_strategy FROM transfers WHERE transfer_type = 'Recurring' ORDER BY vega_time DESC LIMIT 1
),
game_rewards as (
  select r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, 'ENTITY_SCOPE_TEAMS' as entity_scope
  from rewards r
  join game_epochs ge on r.game_id = ge.game_id and r.epoch_id = ge.epoch_id
  JOIN dispatch_strategies t ON r.game_id = t.game_id AND t.dispatch_strategy ->> 'entity_scope' = '2'
  join team_members tm on r.party_id = tm.party_id
  left join game_team_rankings tr on r.game_id = tr.game_id and r.epoch_id = tr.epoch_id and tm.team_id = tr.team_id
  left join game_team_member_rankings tmr on r.game_id = tmr.game_id and r.epoch_id = tmr.epoch_id and tm.team_id = tmr.team_id and r.party_id = tmr.party_id
  where dispatch_strategy->>'entity_scope' = '2'
  union all
  select r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
  from rewards r
  join game_epochs ge on r.game_id = ge.game_id and r.epoch_id = ge.epoch_id
  JOIN dispatch_strategies t ON r.game_id = t.game_id AND dispatch_strategy ->> 'entity_scope' = '1'
  left join game_individual_rankings tmr on r.game_id = tmr.game_id and r.epoch_id = tmr.epoch_id and r.party_id = tmr.party_id
  where dispatch_strategy->>'entity_scope' = '1'
)
select *
from game_rewards
;

-- +goose Down

create or replace view game_stats as
with game_epochs as (
  select distinct game_id, epoch_id
  from rewards
  where game_id is not null
), game_rewards as (
  select r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, tr.total_rewards as team_total_rewards, 'ENTITY_SCOPE_TEAMS' as entity_scope
  from rewards r
  join game_epochs ge on r.game_id = ge.game_id and r.epoch_id = ge.epoch_id
  join transfers t on r.game_id = t.game_id and t.transfer_type = 'Recurring'
  join team_members tm on r.party_id = tm.party_id
  left join game_team_rankings tr on r.game_id = tr.game_id and r.epoch_id = tr.epoch_id and tm.team_id = tr.team_id
  left join game_team_member_rankings tmr on r.game_id = tmr.game_id and r.epoch_id = tmr.epoch_id and tm.team_id = tmr.team_id and r.party_id = tmr.party_id
  where dispatch_strategy->>'entity_scope' = '2'
  union all
  select r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, null as team_total_rewards,'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
  from rewards r
  join game_epochs ge on r.game_id = ge.game_id and r.epoch_id = ge.epoch_id
  join transfers t on r.game_id = t.game_id and t.transfer_type = 'Recurring'
  left join game_individual_rankings tmr on r.game_id = tmr.game_id and r.epoch_id = tmr.epoch_id and r.party_id = tmr.party_id
  where dispatch_strategy->>'entity_scope' = '1'
)
select *
from game_rewards
;

create or replace view game_stats_current as
with game_epochs as (
    select game_id, max(epoch_id) as epoch_id
    from rewards
    where game_id is not null
    group by game_id
), game_rewards as (
  select r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, 'ENTITY_SCOPE_TEAMS' as entity_scope
  from rewards r
  join game_epochs ge on r.game_id = ge.game_id and r.epoch_id = ge.epoch_id
  join transfers t on r.game_id = t.game_id and t.transfer_type = 'Recurring'
  join team_members tm on r.party_id = tm.party_id
  left join game_team_rankings tr on r.game_id = tr.game_id and r.epoch_id = tr.epoch_id and tm.team_id = tr.team_id
  left join game_team_member_rankings tmr on r.game_id = tmr.game_id and r.epoch_id = tmr.epoch_id and tm.team_id = tmr.team_id and r.party_id = tmr.party_id
  where dispatch_strategy->>'entity_scope' = '2'
  union all
  select r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
  from rewards r
  join game_epochs ge on r.game_id = ge.game_id and r.epoch_id = ge.epoch_id
  join transfers t on r.game_id = t.game_id and t.transfer_type = 'Recurring'
  left join game_individual_rankings tmr on r.game_id = tmr.game_id and r.epoch_id = tmr.epoch_id and r.party_id = tmr.party_id
  where dispatch_strategy->>'entity_scope' = '1'
)
select *
from game_rewards
;
