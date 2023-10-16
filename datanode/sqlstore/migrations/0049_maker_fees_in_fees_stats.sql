-- +goose Up

alter table fees_stats
    add column total_maker_fees_received jsonb not null default '[]';

alter table fees_stats
    add column maker_fees_generated jsonb not null default '[]';


-- +goose Down

alter table fees_stats
    drop column total_maker_fees_received;

alter table fees_stats
    drop column maker_fees_generated;

