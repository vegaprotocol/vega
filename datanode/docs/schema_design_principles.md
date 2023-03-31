# Schema Design Principles

Our database is generally 'insert only', so when a particular entity changes state, we *add* a new row in the database, we don't update an existing one.

This is important for both being able to audit what happened in the past, and also for the functioning of decentralised history

The way this is implemented is that most tables have a `Foo` *table* and a `FooCurrent` *view*, where `Foo` has every single change and `FooCurrent` is a database view which selects the most recent version.

For example with proposals we have a table:

```sql
CREATE TABLE proposals(
  id                        BYTEA NOT NULL,
  state                     proposal_state NOT NULL,
  vega_time                 TIMESTAMP WITH TIME ZONE NOT NULL,
  ....
)
```

And a corresponding view

```
CREATE VIEW proposals_current AS (
  SELECT DISTINCT ON (id) * FROM proposals ORDER BY id, vega_time DESC
);
```

So for a proposal with id 0x01 that was initially `STATE_OPEN`  and then `STATE_PASSED`  in the `proposals` table we might have

| id | state | vega_time |
| --- | --- | --- |
| 0x01   | STATE_OPEN    | 2023-01-01 00:00 |
| 0x01   | STATE_PASSED  | 2023-01-02 00:00 |

But selecting from `proposals_current` you would receive, for each distinct ID, the row with the highest `vega_time`:

| id | state | vega_time |
| --- | --- | --- |
| 0x01   | STATE_PASSED  | 2023-01-02 00:00 |

## Performance Implications

The design above is great for insert performance, because we don’t have to go looking for rows to update when changes come along. However, without a bit of care can have quite a surprising effect on the performance when querying the `xxx_current` views.

If you are filtering on a field that is not part of the `DISTINCT ON(yyy)` clause, the database has no choice but to materialise the entire view before querying it.

For example, if you were to query

```sql
SELECT * FROM proposals_current WHERE state='STATE_OPEN'
```

The only way that can be determined is to 

- First get the current version of each proposal (e.g. the set of rows with unique `id`s corresponding to the highest `vega_time` for each `id`)
- Then apply your filter on `state` to that materialized view.

Postgres **cannot reverse** the order of those operations because otherwise you would be asking the question **

> *show me the highest version of a proposal in which it’s state was `OPEN`*
> 

instead of the intended

> *show me, out of the most recent versions of all proposals, which of those has state `OPEN`*
> 

If the size of that materialised view is large, even though your subsequent filtering might well reduce it to a handful of rows, the query will be slow. 

Additionally the materialization of the view doesn’t have any indexes on it.

### Workaround #1 - DISTINCT ON

If the column being filtered is part of the the `DISTINCT ON` clause Postgres ***can*** push the filter down into the table scan. For example this is doesn’t require materializing the whole view:

```sql
SELECT * FROM proposals_current WHERE id='\x01'
```

Because `id` is part of the `DISTINCT ON` clause, it is safe for Postgres to apply your filter to the table itself rather than on the result of the view. It can use any relevant indexes on the table to do that quickly.

If you are sure that a particular column will never change for a particular `id` , you can add it to the `DISTINCT ON` clause of the view - for example:

```sql
CREATE VIEW proposals_current AS (
  SELECT DISTINCT ON (id, reference) * FROM proposals ORDER BY id, reference, vega_time DESC
);
```

And that will greatly improve the performance of a query like

```sql
SELECT * FROM proposals_current WHERE reference='foo'
```

### Workaround #2- Temporal tables maintained by triggers

If you want speedy queries on large tables where you want to filter by columns that **do** change, there is no choice but to make your inserts do some updating as well. 

We do this on our orders table. It’s set up like this:

```sql
CREATE TABLE orders (
    id                BYTEA                    NOT NULL,
    vega_time         TIMESTAMP WITH TIME ZONE NOT NULL,
    current           BOOLEAN NOT NULL DEFAULT FALSE,
);
```

The `current` field is the extra special sauce.

- For row corresponding to the most recent update to an order, it is always `true`.
- For a row corresponding to a previous state of an order, it is set to `false`
- A trigger handles updating of `current` on the previous most current previous version when an insert is made to the table

This allows us to now create our `orders_current` view as follows:

```sql
CREATE VIEW orders_current AS (
  SELECT * FROM orders WHERE current = true
);
```

This view doesn’t suffer from the same issues of not being able to push down filters into the table scan that views created with `DISTINCT ON` has. You can even create indexes that only contain rows where `current=true`:

```sql
CREATE INDEX ON orders (market_id) where current=true;
```

Which keeps the size of the index down and makes querying the latest version of orders by `market_id` as fast as if the table didn’t have all the historical versions in there as well.

**Downsides**

- `INSERT` becomes considerably slower, and `UPDATE` of existing rows generate dead tuples in the database that cause bloat and need to be `VACUUM`-ed later.
- Network history has to take into account this flag and ensure that after restoring from history segments the table is processed such that only the latest 
- version of the order is market as snapshots and sharing, and requires special handling there to work.

### Workaround #3 - Separate ‘audit’ tables

The idea here is to keep a table with only the most recent versions of things, and a separate table with all this changes. This is quite a common pattern, but we currently don’t do this as we weren’t able to get good enough insert performance.