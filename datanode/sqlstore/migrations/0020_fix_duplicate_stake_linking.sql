-- +goose Up

delete from stake_linking where id='\x6fb63c814ffe23b706decf5aeb0be88727b19618970655d5257c189454b4520f';
delete from stake_linking_current where id='\x6fb63c814ffe23b706decf5aeb0be88727b19618970655d5257c189454b4520f';