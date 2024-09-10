## Verifying positions and PnL

At its heart, we are testing a trading platform. Therefore, we need to have the ability to verify the positions held by traders, and their realised or unrealised profit and loss (PnL). The data-node component implements a positions API which collates data events sent out when parties trade, positions are marked to market or settled, when perpetual markets process a funding payment, etc...
The integration test framework implements a number of steps to verify whether or not a trade took place, verifying [individual transfer data](transfers.md). In addition to this, the integration test framework implements a step that allows us to verify the position data analogous to the data-node API.

```cucumber
Then the parties should have the following profit and loss:
  | market id | party      | volume | unrealised pnl | realised pnl | status                        | taker fees | taker fees since | maker fees | maker fees since | other fees | other fees since | funding payments | funding payments since |
  | ETH/DEC19 | trader2    | 0      | 0              | 0            | POSITION_STATUS_ORDERS_CLOSED | 0          | 0                | 0          | 0                | 0          | 0                | 0                | 0                      |
  | ETH/DEC19 | trader3    | 0      | 0              | -162         | POSITION_STATUS_CLOSED_OUT    | 0          | 0                | 0          | 0                | 0          | 0                | 0                | 0                      |
  | ETH/DEC19 | auxiliary1 | -10    | -900           | 0            |                               | 0          | 0                | 0          | 0                | 0          | 0                | 0                | 0                      |
  | ETH/DEC19 | auxiliary2 | 5      | 475            | 586          |                               | 0          | 0                | 0          | 0                | 1          | 0                | 0                | 0                      |
 ```

 Where the fields are defined as follows:

 ```
 | name                   | type           | required |
 |------------------------|----------------|----------|
 | market id              | string         | no       |
 | party                  | string         | yes      |
 | volume                 | int64          | yes      |
 | unrealised pnl         | Int            | yes      |
 | realised pnl           | Int            | yes      |
 | status                 | PositionStatus | no       |
 | taker fees             | Uint           | no       |
 | maker fees             | Uint           | no       |
 | other fees             | Uint           | no       |
 | taker fees since       | Uint           | no       |
 | maker fees since       | Uint           | no       |
 | other fees since       | Uint           | no       |
 | funding payments       | Int            | no       |
 | funding payments since | Int            | no       |
 | is amm                 | bool           | no       |
 ```

Details for the [`PositionStatus` type](types.md#Position-status)

