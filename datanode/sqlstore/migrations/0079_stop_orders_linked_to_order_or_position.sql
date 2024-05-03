-- +goose Up

alter table stop_orders
    add column if not exists size_override_setting int not null default 0,
    add column if not exists size_override_value varchar null;


