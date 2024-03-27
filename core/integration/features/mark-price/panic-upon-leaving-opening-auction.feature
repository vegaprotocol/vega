Feature: Test setting of first mark price with bound violation
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.99        | 3                 |
    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | prices.ETH.value | TYPE_INTEGER | 0              |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 | market type |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 5                | default-none | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 0.1          | 0.5         | 500000      | 0,0,1,0        | 0s,0s,24h0m0s,0s           | oracle1 | future      |

  Scenario: when mark price methodology hasn't provided a price at the point of uncrossing the opening auction (0009-MRKP-001,0009-MRKP-003)
    Given the parties deposit on asset's general account the following amount:
      | party             | asset | amount       |
      | buySideProvider   | USD   | 100000000000 |
      | sellSideProvider  | USD   | 100000000000 |
      | lp1               | USD   | 100000000000 |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1    | ETH/FEB23 | 50000             | 0.001 | submission |
      | lp1 | lp1    | ETH/FEB23 | 50000             | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume  | offset |
      | lp1   | ETH/FEB23 | 49        | 1                    | sell | ASK              | 49      | 20     |
      | lp1   | ETH/FEB23 | 52        | 1                    | buy  | BID              | 52      | 20     |
    And the parties place the following orders:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     | reference    |
      | lp1               | ETH/FEB23 | buy  | 1      | 15899  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | lp1               | ETH/FEB23 | sell | 1      | 15901  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | buySideProvider   | ETH/FEB23 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |              |
      | buySideProvider   | ETH/FEB23 | buy  | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | sellSideProvider  | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | sellSideProvider  | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |              |

    When the network moves ahead "1" blocks
    Then the mark price should be "0" for the market "ETH/FEB23"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/FEB23"

    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 16000 | -1s         |
    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                 | horizon | min bound | max bound |
      | 0          | TRADING_MODE_OPENING_AUCTION | 5       | 15801     | 15999     |


    When the network moves ahead "5" blocks
    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | horizon | min bound | max bound |
    # | 16000      | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 5       | 15900     | 16100     |
      | 0          | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE       |         |           |           |
    And the parties place the following orders with ticks:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     | reference    |
      | buySideProvider   | ETH/FEB23 | buy  | 1      | 14000  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | sellSideProvider  | ETH/FEB23 | sell | 1      | 14000  | 0                | TYPE_LIMIT | TIF_GTC |              |
    