## Governance placeholders

The integration test framework does not bootstrap the governance engine, but rather replaces it. It submits proposals directly to the execution engine as though it were a proposal that has gone through governance. This document will cover these quasi governance steps to:

- [Update a market](#Updating-markets)
- [De-/Re-activate markets](#Governance-auctions)
  - [Suspend a market](#Suspending-markets)
  - [Resume a market](#Resuming-markets)
- [Terminate a market](#Terminating-markets)
- [Set or change network parameters](#Network-parameters)
- [Update assets](#Updating-assets)

### Updating markets

Markets can be updated throughout, some of the paramters that can be changed (things like price monitoring) require setting up new price monitoring paramters (or using a default). How this can be done is outlined in the documentation detailing [setting up markets](markets.md).

```cucumber
When the markets are updated:
  | id        | price monitoring | linear slippage factor | sla params | liquidity fee settings | risk model   | liquidation strategy  |
  | ETH/MAR22 | new-pm           | 1e-3                   | new-sla    | new-fee-conf           | new-risk-mdl | new-liquidation-strat |
```

All fields, bar the ID are treated as optional, and are defined as follows:

```
| name                       | type                                 | NOTE                                            |
| id                         | string (market ID)                   |                                                 |
| linear slippage factor     | float64                              |                                                 |
| quadratic slippage factor  | float64                              | deprecated                                      |
| data source config         | string (oracle name)                 | not possible to update the product in all cases |
| price monitoring           | string (price monitoring name)       |                                                 |
| risk model                 | string (risk model name)             |                                                 |
| liquidity monitoring       | string (liquidity monitoring name)   | deprecated                                      |
| sla params                 | string (sla params name)             |                                                 |
| liquidity fee settings     | string (fee config name)             |                                                 |
| liquidation strategy       | string (liquidation strategy name)   |                                                 |
| price type                 | Price_Type                           |                                                 |
| decay weight               | decimal                              |                                                 |
| decay power                | decimal                              |                                                 |
| cash amount                | Uint                                 |                                                 |
| source weights             | Source_Weights                       |                                                 |
| source staleness tolerance | Staleness_Tolerance                  |                                                 |
| oracle1                    | string (composite price oracle name) |                                                 |
| oracle2                    | string (composite price oracle name) |                                                 |
| oracle3                    | string (composite price oracle name) |                                                 |
| oracle4                    | string (composite price oracle name) |                                                 |
| oracle5                    | string (composite price oracle name) |                                                 |
| tick size                  | uint                                 |                                                 |
```

Details on the [`Price_Type` type](types.md#Price-type).
Details on the [`Source_Weights` type](types.md#Source-weights)
Details on the [`Staleness_Tolerance` type](types.md#Staleness-tolerance)

Any field that is not set means that aspect of the market configuration is not to be updated.

### Governance auctions

Markets can be put into governance auctions, which can be ended through governance, too.

#### Suspending markets

To start a governance auction, the following step is used:

```cucumber
When the market states are updated through governance:
  | market id | state                            |
  | ETH/DEC20 | MARKET_STATE_UPDATE_TYPE_SUSPEND |
```

Where the relevant fields are:

```
| market id | string (market ID) | required |
| state     | MarketStateUpdate  | required |
| error     | expected error     | optional |
```

Details on the [`MarketStateUpdate` type](types.md#Market-state-update)

#### Resuming markets

To end a goverance auction, the same step is used like so:

```cucumber
When the market states are updated through governance:
  | market id | state                           |
  | ETH/DEC20 | MARKET_STATE_UPDATE_TYPE_RESUME |
```

Where the relevant fields are:

```
| market id | string (market ID) | required |
| state     | MarketStateUpdate  | required |
| error     | expected error     | optional |
```

Details on the [`MarketStateUpdate` type](types.md#Market-state-update)

### Terminating markets

A market can be terminated through governace, too. This can be done with or without a settlement price:

```cucumber
When the market states are updated through governance:
  | market id | state                              | settlement price |
  | ETH/DEC19 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 976              |
```

Where the relevant fields are:

```
| market id        | string (market ID) | required |
| state            | MarketStateUpdate  | required |
| settlement price | Uint               | optional |
| error            | expected error     | optional |
```

Details on the [`MarketStateUpdate` type](types.md#Market-state-update)

### Network parameters

Setting network parameters is typically done as part of the `Background` part of a feature file, or at the start of a scenario. However, changing some network paramters may have an effect on active markets. In that case, a transaction that failed or succeeded previously is expected to behave differently after the network parameters have been updated. This can be useful to test whether or not network paramter changes are correctly propagated. Setting or updating network paramters is done using this step:

```cucumber
Background:
  # setting network parameter to an initial value
  Given the following network parameters are set:
    | name                           | value |
    | limits.markets.maxPeggedOrders | 2     |

Scenario:
  When the parties place the following pegged orders:
    | party  | market id | side | volume | pegged reference | offset | error                         |
    | party1 | ETH/DEC24 | buy  | 100    | BEST_BID         | 5      |                               |
    | party1 | ETH/DEC24 | buy  | 200    | BEST_BID         | 10     |                               |
    | party1 | ETH/DEC24 | buy  | 250    | BEST_BID         | 15     | error: too many pegged orders |

  Then the following network parameters are set:
    | name                           | value |
    | limits.markets.maxPeggedOrders | 2     |

  When the parties place the following pegged orders:
    | party  | market id | side | volume | pegged reference | offset | error |
    | party1 | ETH/DEC24 | buy  | 250    | BEST_BID         | 15     |       |
```

_Note: the error is not necessarily the correct value._


The fields are both required and defined as follows:

```
| name  | string (the network paramter key name)              |
| value | string (must be compatible with the parameter type) |
```

### Updating assets

Similarly to registering assets, it is possible to re-define an existing asset, though this is a rather niche thing to do, using this step:

```cucumber
When the following assets are updated:
  | id  | decimal places | quantum |
  | BTC | 5              | 20      |
```

Where the fields are defined as follows:

```
| id             | string  | required |
| decimal places | uint64  | required |
| quantum        | decimal | optional |
```

Should this cause an error, the test will fail.
