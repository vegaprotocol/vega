-- +goose Up

-- create the history and current tables for game team scores
create table if not exists game_team_scores (
       game_id bytea not null,
       team_id bytea not null,
       epoch_id bigint not null,
       score NUMERIC NOT NULL,
       vega_time timestamp with time zone not null,
       primary key (vega_time, game_id, team_id)
);

create table if not exists game_party_scores (
       game_id bytea not null,
       party_id bytea not null,
       team_id bytea,
       epoch_id bigint not null,
       score NUMERIC NOT NULL,
       staking_balance HUGEINT,    
	open_volume HUGEINT,
       total_fees_paid HUGEINT not null,
       is_eligible boolean,
       rank integer,
       vega_time timestamp with time zone not null,
       primary key (vega_time, game_id, party_id)
);


select create_hypertable('game_team_scores', 'vega_time', chunk_time_interval => INTERVAL '1 day');
select create_hypertable('game_party_scores', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table if not exists game_team_scores_current (
       game_id bytea not null,
       team_id bytea not null,
       epoch_id bigint not null,
       score NUMERIC NOT NULL,
       vega_time timestamp with time zone not null,
       primary key (game_id, team_id)
);

create table if not exists game_party_scores_current (
       game_id bytea not null,
       party_id bytea not null,
       team_id bytea,
       epoch_id bigint not null,
       score NUMERIC NOT NULL,
       staking_balance HUGEINT,    
       open_volume HUGEINT,
       total_fees_paid HUGEINT not null,
       is_eligible boolean,
       rank integer,
       vega_time timestamp with time zone not null,
       primary key (game_id, party_id)
);

-- create the trigger functions and triggers
-- +goose StatementBegin
create or replace function update_game_team_scores()
       returns trigger
       language plpgsql
as $$
   begin
        insert into game_team_scores_current(game_id,team_id, epoch_id, score, vega_time)
        values (new.game_id, new.team_id, new.epoch_id, new.score, new.vega_time)
        on conflict(game_id, team_id)
        do update set
           epoch_id = excluded.epoch_id,
           score = excluded.score,
	       vega_time = excluded.vega_time;
        return null;
   end;
$$;

create or replace function update_game_party_scores()
       returns trigger
       language plpgsql
as $$
   begin
        insert into game_party_scores_current(game_id,party_id, team_id, epoch_id, score, 
        staking_balance,open_volume,total_fees_paid,is_eligible,vega_time)
        values (new.game_id, new.party_id, new.team_id, new.epoch_id, new.score, 
                new.staking_balance, new.open_volume, new.total_fees_paid,
                new.is_eligible,new.vega_time)
        on conflict(game_id, party_id)
        do update set
           epoch_id = excluded.epoch_id,
           team_id = excluded.team_id,
           score = excluded.score,
           staking_balance = excluded.staking_balance,
           open_volume = excluded.open_volume,
           total_fees_paid = excluded.total_fees_paid,
           is_eligible = excluded.is_eligible,
           vega_time = excluded.vega_time;
        return null;
   end;
$$;
-- +goose StatementEnd

create trigger update_game_team_scores
    after insert or update
    on game_team_scores
    for each row execute function update_game_team_scores();

create trigger update_game_party_scores
    after insert or update
    on game_party_scores
    for each row execute function update_game_party_scores();

-- +goose Down

drop table if exists game_team_scores_current;
drop table if exists game_team_scores;
drop function if exists update_game_team_scores;

drop table if exists game_party_scores_current;
drop table if exists game_party_scores;
drop function if exists update_game_party_scores;