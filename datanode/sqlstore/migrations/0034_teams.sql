-- +goose Up

create table if not exists teams
(
    id               bytea                    not null,
    referrer         bytea                    not null, -- ID of the party that created the team
    name             varchar                  not null,
    team_url         varchar,
    avatar_url       varchar,
    closed           BOOLEAN                  NOT NULL DEFAULT FALSE,
    created_at_epoch bigint                   not null,
    created_at       timestamp with time zone not null,
    vega_time        timestamp with time zone not null,
    primary key (id)
);

create table if not exists team_members
(
    team_id         bytea                    not null references teams (id),
    party_id        bytea                    not null, -- ID of the party that joined the team
    joined_at_epoch bigint                   not null,
    joined_at       timestamp with time zone not null,
    vega_time       timestamp with time zone not null,
    primary key (team_id, party_id, joined_at_epoch)
);

create view current_team_members as
select distinct on (party_id) *
from team_members
order by party_id, joined_at_epoch desc;

-- +goose Down

drop view if exists team_members_history;
drop view if exists current_team_members;
drop table if exists team_members;
drop table if exists teams;
