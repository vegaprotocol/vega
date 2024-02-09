-- +goose Up
alter table game_reward_totals
    add column if not exists total_rewards_quantum hugeint not null default 0;

-- Postgres is throwing hissy fits when we try to replace these views so we are forced to drop them
-- and then recreate them the way we want them

-- as game_stats and game_stats_current has a dependency on game_team_rankings, we have to drop them
-- first or we won't be able to drop game_team_rankings
drop view if exists game_stats;
drop view if exists game_stats_current;
drop view if exists game_team_rankings;

-- we need to drop these two as they have a dependency on game_team_rankings
create view game_team_rankings as
with team_games as (
    -- get the games where the entity scope is individuals
    select distinct game_id from transfers
    where dispatch_strategy ->> 'entity_scope' = '2'
      and game_id is not null
), team_totals as (
    select t.game_id,
           t.asset_id,
           t.epoch_id,
           t.team_id,
           sum(t.total_rewards) as total_rewards,
           sum(t.total_rewards_quantum) as total_rewards_quantum
    from game_reward_totals t
    join team_games g on t.game_id = g.game_id
    where t.team_id != '\x'
    group by t.game_id, t.asset_id, t.epoch_id, t.team_id
)
    select game_id,
           epoch_id,
           team_id,
           total_rewards,
           total_rewards_quantum,
           rank() over (
               partition by game_id, epoch_id order by total_rewards_quantum desc
           ) as rank
           from team_totals;

drop view if exists game_team_member_rankings;
create view game_team_member_rankings as
with team_games as (
    -- get the games where the entity scope is individuals
    select distinct game_id from transfers
    where dispatch_strategy ->> 'entity_scope' = '2'
      and game_id is not null
), team_totals as (
    select t.game_id, t.asset_id, t.team_id, t.party_id, t.epoch_id, sum(t.total_rewards) as total_rewards, sum(t.total_rewards_quantum) as total_rewards_quantum
    from game_reward_totals t
             join team_games g on t.game_id = g.game_id
    where t.team_id != '\x'
    group by t.game_id, t.asset_id, t.team_id, t.party_id, t.epoch_id
)
    select game_id,
           epoch_id,
           team_id,
           party_id,
           total_rewards,
           total_rewards_quantum,
           rank() over (
               partition by game_id, epoch_id, team_id order by total_rewards_quantum desc
           ) as rank
    from team_totals;

drop view if exists game_individual_rankings;
create view game_individual_rankings as
with individual_games as (
    -- get the games where the entity scope is individuals
    select game_id from transfers
    where dispatch_strategy ->> 'entity_scope' = '1'
      and game_id is not null
), individual_totals as (
    -- calculate the total rewards for each individual in each individual entity scoped game
    select t.game_id, t.epoch_id, t.asset_id, t.party_id, sum(t.total_rewards) as total_rewards, sum(total_rewards_quantum) as total_rewards_quantum
    from game_reward_totals t
             join individual_games i on t.game_id = i.game_id
    group by t.game_id, t.epoch_id, t.asset_id, t.party_id
), individual_rankings as (
-- rank the individuals for each game at each epoch
    select game_id,
           epoch_id,
           party_id,
           total_rewards_quantum,
           rank() over (
               partition by game_id, epoch_id order by total_rewards_quantum desc
           ) as rank
    from individual_totals
)
    select it.game_id,
           it.epoch_id,
           it.party_id,
           it.total_rewards,
           ir.total_rewards_quantum,
           ir.rank
    from individual_totals it
    join individual_rankings ir on it.game_id = ir.game_id and it.epoch_id = ir.epoch_id and it.party_id = ir.party_id;

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

create or replace view current_game_reward_totals as (
    with current_game_epochs as (
        select game_id, max(epoch_id) as epoch_id
        from game_reward_totals
        group by game_id
    )
    select grt.*
    from game_reward_totals grt
    join current_game_epochs cge on grt.game_id = cge.game_id and grt.epoch_id = cge.epoch_id
);

-- +goose Down

drop view if exists game_stats;
drop view if exists game_stats_current;
drop view if exists game_team_rankings;

create view game_team_rankings as
with team_games as (
    -- get the games where the entity scope is individuals
    select distinct game_id from transfers
    where dispatch_strategy ->> 'entity_scope' = '2'
      and game_id is not null
), team_totals as (
    select t.game_id, t.asset_id, t.epoch_id, t.team_id, sum(t.total_rewards) as total_rewards
    from game_reward_totals t
             join team_games g on t.game_id = g.game_id
    where t.team_id != '\x'
    group by t.game_id, t.asset_id, t.epoch_id, t.team_id
)
select game_id, epoch_id, team_id, total_rewards, rank() over (partition by game_id, epoch_id order by total_rewards desc) as rank
from team_totals;

drop view if exists game_team_member_rankings;
create view game_team_member_rankings as
with team_games as (
    -- get the games where the entity scope is individuals
    select distinct game_id from transfers
    where dispatch_strategy ->> 'entity_scope' = '2'
      and game_id is not null
), team_totals as (
    select t.game_id, t.asset_id, t.team_id, t.party_id, t.epoch_id, sum(t.total_rewards) as total_rewards
    from game_reward_totals t
             join team_games g on t.game_id = g.game_id
    where t.team_id != '\x'
    group by t.game_id, t.asset_id, t.team_id, t.party_id, t.epoch_id
)
select game_id, epoch_id, team_id, party_id, total_rewards, rank() over (partition by game_id, epoch_id, team_id order by total_rewards desc) as rank
from team_totals;

drop view if exists game_individual_rankings;
create view game_individual_rankings as
with individual_games as (
    -- get the games where the entity scope is individuals
    select game_id from transfers
    where dispatch_strategy ->> 'entity_scope' = '1'
      and game_id is not null
), individual_totals as (
    -- calculate the total rewards for each individual in each individual entity scoped game
    select t.game_id, t.epoch_id, t.asset_id, t.party_id, sum(t.total_rewards) as total_rewards
    from game_reward_totals t
             join individual_games i on t.game_id = i.game_id
    group by t.game_id, t.epoch_id, t.asset_id, t.party_id
)
-- rank the individuals for each game at each epoch
select game_id, epoch_id, party_id, total_rewards, rank() over (partition by game_id, epoch_id order by total_rewards desc) as rank
from individual_totals;

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
            r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, tr.total_rewards as team_total_rewards, 'ENTITY_SCOPE_TEAMS' as entity_scope
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
            r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, null as team_total_rewards, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
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
            r.*, t.dispatch_strategy, tm.team_id, tmr.rank as member_rank, tr.rank as team_rank, tmr.total_rewards, 'ENTITY_SCOPE_TEAMS' as entity_scope
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
            r.*, t.dispatch_strategy, null as team_id, tmr.rank as member_rank, null as team_rank, tmr.total_rewards, 'ENTITY_SCOPE_INDIVIDUALS' as entity_scope
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

create or replace view current_game_reward_totals as (
     with current_game_epochs as (
         select game_id, max(epoch_id) as epoch_id
         from game_reward_totals
         group by game_id
     )
     select grt.*
     from game_reward_totals grt
     join current_game_epochs cge on grt.game_id = cge.game_id and grt.epoch_id = cge.epoch_id
);
