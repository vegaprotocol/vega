Feature: Test setting of mark price
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 0.1          | 0.5         | 500000      | 0,1,0,0        | 3h0m0s,2s,24h0m0s,1h25m0s  |

  @SLABug
  Scenario: 001 when network.markPriceUpdateMaximumFrequency=0s
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference    |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 14900  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | party            | ETH/FEB23 | sell | 2      | 15902  | 0                | TYPE_LIMIT | TIF_GTC | party-sell-2 |
      | party            | ETH/FEB23 | sell | 1      | 15904  | 0                | TYPE_LIMIT | TIF_GTC | party-sell-3 |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |              |
      | sellSideProvider | ETH/FEB23 | sell | 3      | 100100 | 0                | TYPE_LIMIT | TIF_GTC |              |

    # AC 0009-MRKP-001, If a market mark price is configured in such a way that the mark price methodology hasn't provided a price at the point of uncrossing the opening auction, then the auction uncrossing price is set as the first mark price, regardless of what the mark price methodology says.
    When the network moves ahead "1" blocks
    Then the mark price should be "0" for the market "ETH/FEB23"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/FEB23"

    When the network moves ahead "1" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

