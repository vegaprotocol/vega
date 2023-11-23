-- +goose Up

alter table rewards
    add column if not exists quantum_amount HUGEINT null;

-- +goose StatementBegin
-- This computes the quantum_amount for old data.
do
$$
    begin
        update rewards
        set quantum_amount = (select assets_current.quantum * rewards.amount from assets_current where assets_current.id = rewards.asset_id);

        alter table rewards
            alter column quantum_amount set not null;

    end
$$;
-- +goose StatementEnd


-- +goose Down

alter table rewards
    drop column if exists quantum_amount;
