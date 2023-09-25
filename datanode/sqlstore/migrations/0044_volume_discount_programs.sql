-- +goose Up

-- create a new volume discount record when a new volume discount program is created,
-- updated, or ended so that we keep an audit trail, just in case.
-- We create it as a hypertable and set a retention policy to make sure
-- old and redundant data is removed in due course.
create table if not exists volume_discount_programs
(
    id                       bytea                    not null,
    version                  int                      not null,
    benefit_tiers            jsonb,
    end_of_program_timestamp timestamp with time zone not null,
    window_length            int                      not null,
    vega_time                timestamp with time zone not null,
    ended_at                 timestamp with time zone,
    seq_num                  bigint                   not null,
    primary key (vega_time, seq_num)
);

select create_hypertable('volume_discount_programs', 'vega_time', chunk_time_interval => INTERVAL '1 day');

-- simplify volume discount retrieval using a view that provides the latest volume discount information.
create view current_volume_discount_program as
(
select *
from volume_discount_programs
order by vega_time desc, seq_num desc
limit 1 -- there should only be 1 volume discount program running at any time, so just get the last record.
    );

create table volume_discount_stats
(
    at_epoch                      bigint                   not null,
    parties_volume_discount_stats jsonb                    not null,
    vega_time                     timestamp with time zone not null,
    primary key (at_epoch, vega_time)
);

select create_hypertable('volume_discount_stats', 'vega_time', chunk_time_interval => INTERVAL '1 day');

-- +goose Down

drop table if exists volume_discount_stats;
drop view if exists current_volume_discount_program;
drop table if exists volume_discount_programs;
