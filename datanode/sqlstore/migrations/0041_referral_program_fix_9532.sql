-- +goose Up

-- first drop the existing primary key
alter table referral_programs drop constraint referral_programs_pkey;

-- add the sequence number column to the table
-- we can just set the default to 0 for any existing rows as previously it would only support
-- one row per vega_time, but there is an edge case where it is possible for the program to start
-- and end in the same block if it has been set up incorrectly, so we need to support this edge case
-- and not fail
alter table referral_programs add column if not exists seq_num bigint not null default 0;

-- now add the new primary key to the table
alter table referral_programs add constraint referral_programs_pkey primary key (vega_time, seq_num);

-- update the current_referral_program view to correctly display the most recent referral program update

-- make sure that it doesn't exist in case migrations have failed previously
drop view if exists current_referral_program;

-- simplify referral retrieval using a view that provides the latest referral information.
create view current_referral_program as (
    select *
    from referral_programs
    order by vega_time desc, seq_num desc
    limit 1 -- there should only be 1 referral program running at any time, so just get the last record.
);

-- +goose Down

-- make sure that it doesn't exist in case migrations have failed previously
drop view if exists current_referral_program;

alter table referral_programs drop constraint referral_programs_pkey;

alter table referral_programs drop column seq_num;

-- simplify referral retrieval using a view that provides the latest referral information.
create view current_referral_program as (
    select *
    from referral_programs
    order by vega_time desc limit 1 -- there should only be 1 referral program running at any time, so just get the last record.
);

alter table referral_programs add constraint referral_programs_pkey primary key (vega_time);
