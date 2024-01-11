-- +goose Up

alter table transfers add column if not exists game_id bytea null;
alter table rewards add column if not exists game_id bytea null;

create table game_reward_totals (
    game_id bytea not null,
    party_id bytea not null,
    asset_id bytea not null,
    market_id bytea not null,
    epoch_id bigint not null,
    team_id bytea not null,         -- participant may take part in a game as a team member or as an individual depending on the entity scope of the game
    total_rewards hugeint not null,
    primary key (game_id, party_id, asset_id, market_id, epoch_id, team_id)
);

-- +goose StatementBegin
create or replace function insert_game_reward_totals()
returns trigger
    language plpgsql
    as $$
    declare party_team_id bytea;
begin
    with current_team_members as (
        select distinct on (party_id) *
        from team_members
        order by party_id, joined_at_epoch desc
    )
    select team_id into party_team_id from current_team_members where party_id = new.party_id;

    update game_reward_totals
    set team_id = coalesce(party_team_id, '\x')
    where game_id = new.game_id and party_id = new.party_id
      and asset_id = new.asset_id and market_id = new.market_id
      and epoch_id = new.epoch_id;

    return null;
end;
$$;
-- +goose StatementEnd

-- we don't know the team_id of the party when the reward is emitted by core
-- so we need to look it up from the team_members table and insert it into the game_reward_totals table
-- when we insert a new total reward for a party
create trigger insert_game_reward_totals after insert on game_reward_totals
    for each row execute procedure insert_game_reward_totals();

create or replace view game_team_rankings as
with team_totals as (
    select game_id, asset_id, epoch_id, team_id, sum(total_rewards) as total_rewards
    from game_reward_totals
    where team_id != '\x'
    group by game_id, asset_id, epoch_id, team_id
)
select game_id, epoch_id, team_id, total_rewards, rank() over (partition by game_id, epoch_id order by total_rewards desc) as rank
from team_totals;

create or replace view game_team_member_rankings as
with team_totals as (
    select game_id, asset_id, team_id, party_id, epoch_id, sum(total_rewards) as total_rewards
    from game_reward_totals
    where team_id != '\x'
    group by game_id, asset_id, team_id, party_id, epoch_id
)
select game_id, epoch_id, team_id, party_id, total_rewards, rank() over (partition by game_id, epoch_id, team_id order by total_rewards desc) as rank
from team_totals;

create or replace view game_individual_rankings as
with individual_totals as (
  select game_id, epoch_id, asset_id, party_id, sum(total_rewards) as total_rewards
  from game_reward_totals
  where team_id = '\x'
  group by game_id, epoch_id, asset_id, party_id
)
select game_id, epoch_id, party_id, total_rewards, rank() over (partition by game_id, epoch_id order by total_rewards desc) as rank
from individual_totals;

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
drop view if exists current_game_reward_totals;
drop view if exists game_stats_current;
drop view if exists game_stats;
drop view if exists game_individual_rankings;
drop view if exists game_team_member_rankings;
drop view if exists game_team_rankings;

drop trigger if exists insert_game_reward_totals on game_reward_totals;
drop function if exists insert_game_reward_totals;
drop table if exists game_reward_totals;

alter table transfers drop column if exists game_id;
alter table rewards drop column if exists game_id;
