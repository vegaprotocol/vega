Feature: Composite mark price calculation

  Background:
    Given the average block duration is "1"
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | price1.USD.value | TYPE_INTEGER | 0              |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 0            | 0           | 100         | 1,1,1,0        | 1m0s,1h0m0s,5m0s,0s        | oracle1 |

  Scenario: Composite price composed of last traded, order book, and oracle price (0009-MRKP-022)(0009-MRKP-023)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14970 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 15090 | 0                | TYPE_LIMIT | TIF_GTC |           |
    When the opening auction period ends for market "ETH/FEB23"
    # (15000+15030)/2=15015
    Then the mark price should be "15015" for the market "ETH/FEB23"

    # No price sources stale, composite price average of last traded price, book price and oracle price
    Given the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | price1.USD.value | 15060 | 0s          |
    When the network moves ahead "2" blocks
    Then the mark price should be "15030" for the market "ETH/FEB23"

    # Last traded price stale, composite price average of book price and oracle price
    When the network moves ahead "61" blocks
    Then the mark price should be "15045" for the market "ETH/FEB23"

    # Last traded and oracle price stale, composite price average of book price
    When the network moves ahead "241" blocks
    Then the mark price should be "15030" for the market "ETH/FEB23"








