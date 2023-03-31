# Orders

These are a bit more complicated because there are two types of updates to orders:

- Updates initiated by the user, by sending an amend transaction
- Updates that happen in core, because the state of the order has change in some way (e.g. due to a fill)

To accommodate this, the order table is unique in having a `version` column *which is incremented only when the user amends his order by submitting a transaction.* It is ******not****** incremented when the state of the order is changed internally by Vega.

```sql
CREATE TABLE orders (
    id                BYTEA                     NOT NULL,
    version           INT                       NOT NULL,
    vega_time         TIMESTAMP WITH TIME ZONE  NOT NULL,
    ....
);
```

Now we have two concepts of *version*:

- The *user* concept of version whereby each time he amends an order the version number goes up.
- The *system* concept of version; the latest of which is the row with the highest `vega_time`

## Worked Example

Lets imagine the `orders` table looks a bit like this:

|  | id | version | status | vega_time |
| --- | --- | --- | --- | --- |
| user submits order | 0x01 | 1 | ACTIVE            | 2023-01-01 00:01 |
| order gets partially matched | 0x01 | 1 | PARTIALLY_FILLED  | 2023-01-01 00:02 |
| user amends order price level | 0x01 | 2 | PARTIALLY_FILLED  | 2023-01-01 00:03 |
| rest of order gets filled | 0x01 | 2 | FILLED            | 2023-01-01 00:04 |

We have three views over this data:

`orders_current`

For each distinct `id`, the set of rows with the highest `vega_time`

| id | version | status | vega_time |
| --- | --- | --- | --- |
| 0x01 | 2 | FILLED            | 2023-01-01 00:04 |

`orders_current_versions`

For each distinct `(id, version)` pair, set of rows in the table with the highest `vega_time`. 

It shows you the latest *system versions* for all the *user amended versions* of an order. 

| id | version | status | vega_time |
| --- | --- | --- | --- |
| 0x01 | 1 | PARTIALLY_FILLED  | 2023-01-01 00:02 |
| 0x01 | 2 | FILLED            | 2023-01-01 00:04 |

`orders_live`

The set of orders which are actively trading or parked. We maintain it as a real table with triggers for performance reasons, but it is equivalent to 

```sql
select * from orders_current where status in (ACTIVE, PARKED)
```