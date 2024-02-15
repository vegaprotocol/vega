-- +goose Up

drop view if exists game_stats;
drop view if exists game_stats_current;

-- These materialized views should only be updated once per epoch
create materialized view game_stats as
with
    game_epochs as (
        select distinct
            game_id, epoch_id
        from rewards
        where
            game_id is not null
    ),
    dispatch_strategies AS (
        SELECT DISTINCT
            ON (game_id) game_id, dispatch_strategy
        FROM transfers
        WHERE
            transfer_type = 'Recurring'
        ORDER BY game_id, vega_time DESC
        ),
        game_rewards as (
            select
                r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
                tr.total_rewards as team_total_rewards, tr.total_rewards_quantum as team_total_rewards_quantum, 'ENTITY_SCOPE_TEAMS' as entity_scope
            from
                rewards r
            join game_epochs ge on r.game_id = ge.game_id
                and r.epoch_id = ge.epoch_id
            JOIN dispatch_strategies t ON r.game_id = t.game_id
                AND t.dispatch_strategy ->> 'entity_scope' = '2'
            join team_members tm on r.party_id = tm.party_id
            left join game_team_rankings tr on r.game_id = tr.game_id
                and r.epoch_id = tr.epoch_id
                and tm.team_id = tr.team_id
            left join game_team_member_rankings tmr on r.game_id = tmr.game_id
                and r.epoch_id = tmr.epoch_id
                and tm.team_id = tmr.team_id
                and r.party_id = tmr.party_id
            union all
            select
                r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
                null as team_total_rewards, null as team_total_rewards_quantum, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
            from
                rewards r
            join game_epochs ge on r.game_id = ge.game_id
                and r.epoch_id = ge.epoch_id
            JOIN dispatch_strategies t ON r.game_id = t.game_id
                AND t.dispatch_strategy ->> 'entity_scope' = '1'
            left join game_individual_rankings tmr on r.game_id = tmr.game_id
                and r.epoch_id = tmr.epoch_id
                and r.party_id = tmr.party_id
        )
select *
from game_rewards
with data;

create materialized view game_stats_current as
with
    game_epochs as (
        select game_id, max(epoch_id) as epoch_id
        from rewards
        where
            game_id is not null
        group by
            game_id
    ),
    dispatch_strategies AS (
        SELECT DISTINCT
        ON (game_id) game_id, dispatch_strategy
        FROM transfers
        WHERE
        transfer_type = 'Recurring'
        ORDER BY game_id, vega_time DESC
        ),
        game_rewards as (
        select
        r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
        tr.total_rewards as team_total_rewards, tr.total_rewards_quantum as team_total_rewards_quantum, 'ENTITY_SCOPE_TEAMS' as entity_scope
        from
        rewards r
        join game_epochs ge on r.game_id = ge.game_id
        and r.epoch_id = ge.epoch_id
        JOIN dispatch_strategies t ON r.game_id = t.game_id
        AND t.dispatch_strategy ->> 'entity_scope' = '2'
        join team_members tm on r.party_id = tm.party_id
        left join game_team_rankings tr on r.game_id = tr.game_id
        and r.epoch_id = tr.epoch_id
        and tm.team_id = tr.team_id
        left join game_team_member_rankings tmr on r.game_id = tmr.game_id
        and r.epoch_id = tmr.epoch_id
        and tm.team_id = tmr.team_id
        and r.party_id = tmr.party_id
        union all
        select
        r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
        null as team_total_rewards, null as team_total_rewards_quantum, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
        from
        rewards r
        join game_epochs ge on r.game_id = ge.game_id
        and r.epoch_id = ge.epoch_id
        JOIN dispatch_strategies t ON r.game_id = t.game_id
        AND t.dispatch_strategy ->> 'entity_scope' = '1'
        left join game_individual_rankings tmr on r.game_id = tmr.game_id
        and r.epoch_id = tmr.epoch_id
        and r.party_id = tmr.party_id
        )
select *
from game_rewards
with data;

-- +goose Down

drop materialized view if exists game_stats;
drop materialized view if exists game_stats_current;

create view game_stats as
with
    game_epochs as (
        select distinct
            game_id, epoch_id
        from rewards
        where
            game_id is not null
    ),
    dispatch_strategies AS (
        SELECT DISTINCT
        ON (game_id) game_id, dispatch_strategy
        FROM transfers
        WHERE
        transfer_type = 'Recurring'
        ORDER BY game_id, vega_time DESC
        ),
        game_rewards as (
        select
        r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
        tr.total_rewards as team_total_rewards, tr.total_rewards_quantum as team_total_rewards_quantum, 'ENTITY_SCOPE_TEAMS' as entity_scope
        from
        rewards r
        join game_epochs ge on r.game_id = ge.game_id
        and r.epoch_id = ge.epoch_id
        JOIN dispatch_strategies t ON r.game_id = t.game_id
        AND t.dispatch_strategy ->> 'entity_scope' = '2'
        join team_members tm on r.party_id = tm.party_id
        left join game_team_rankings tr on r.game_id = tr.game_id
        and r.epoch_id = tr.epoch_id
        and tm.team_id = tr.team_id
        left join game_team_member_rankings tmr on r.game_id = tmr.game_id
        and r.epoch_id = tmr.epoch_id
        and tm.team_id = tmr.team_id
        and r.party_id = tmr.party_id
        union all
        select
        r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
        null as team_total_rewards, null as team_total_rewards_quantum, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
        from
        rewards r
        join game_epochs ge on r.game_id = ge.game_id
        and r.epoch_id = ge.epoch_id
        JOIN dispatch_strategies t ON r.game_id = t.game_id
        AND t.dispatch_strategy ->> 'entity_scope' = '1'
        left join game_individual_rankings tmr on r.game_id = tmr.game_id
        and r.epoch_id = tmr.epoch_id
        and r.party_id = tmr.party_id
        )
select *
from game_rewards;

create view game_stats_current as
with
    game_epochs as (
        select game_id, max(epoch_id) as epoch_id
        from rewards
        where
            game_id is not null
        group by
            game_id
    ),
    dispatch_strategies AS (
        SELECT DISTINCT
        ON (game_id) game_id, dispatch_strategy
        FROM transfers
        WHERE
        transfer_type = 'Recurring'
        ORDER BY game_id, vega_time DESC
        ),
        game_rewards as (
        select
        r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
        tr.total_rewards as team_total_rewards, tr.total_rewards_quantum as team_total_rewards_quantum, 'ENTITY_SCOPE_TEAMS' as entity_scope
        from
        rewards r
        join game_epochs ge on r.game_id = ge.game_id
        and r.epoch_id = ge.epoch_id
        JOIN dispatch_strategies t ON r.game_id = t.game_id
        AND t.dispatch_strategy ->> 'entity_scope' = '2'
        join team_members tm on r.party_id = tm.party_id
        left join game_team_rankings tr on r.game_id = tr.game_id
        and r.epoch_id = tr.epoch_id
        and tm.team_id = tr.team_id
        left join game_team_member_rankings tmr on r.game_id = tmr.game_id
        and r.epoch_id = tmr.epoch_id
        and tm.team_id = tmr.team_id
        and r.party_id = tmr.party_id
        union all
        select
        r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, tmr.total_rewards_quantum,
        null as team_total_rewards, null as team_total_rewards_quantum, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
        from
        rewards r
        join game_epochs ge on r.game_id = ge.game_id
        and r.epoch_id = ge.epoch_id
        JOIN dispatch_strategies t ON r.game_id = t.game_id
        AND t.dispatch_strategy ->> 'entity_scope' = '1'
        left join game_individual_rankings tmr on r.game_id = tmr.game_id
        and r.epoch_id = tmr.epoch_id
        and r.party_id = tmr.party_id
        )
select *
from game_rewards;
