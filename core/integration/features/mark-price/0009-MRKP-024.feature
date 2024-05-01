Feature: Test setting of mark price
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0.1               | 0.9               | ETH        | 18                  |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 360     | 0.9999999   | 10                |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights  | source staleness tolerance | market type |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 1            | 1           | 50000000    | 0,0.999,0.001,0 | 20s,20s,24h0m0s,1h25m0s    | future      |
      | ETH/FEB22 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | perp-oracle            | 0.25                   | 0                         | default-futures | weight     | 1            | 1           | 50000000    | 0,0.999,0.001,0 | 20s,20s,24h0m0s,1h25m0s    | perp        |

  Scenario: 001 check mark price using weight average
    Given the parties deposit on asset's general account the following amount:
      | party             | asset | amount       |
      | buySideProvider   | USD   | 100000000000 |
      | sellSideProvider  | USD   | 100000000000 |
      | party             | USD   | 48050        |
      | buySideProvider1  | USD   | 100000000000 |
      | sellSideProvider1 | USD   | 100000000000 |
      | party1            | USD   | 48050        |
    And the parties place the following orders:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider   | ETH/FEB23 | buy  | 5      | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider   | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider   | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party             | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider  | ETH/FEB23 | sell | 2      | 15920  | 0                | TYPE_LIMIT | TIF_GTC | sell-2    |
      | sellSideProvider  | ETH/FEB23 | sell | 1      | 15940  | 0                | TYPE_LIMIT | TIF_GTC | sell-3    |
      | sellSideProvider  | ETH/FEB23 | sell | 3      | 17000  | 0                | TYPE_LIMIT | TIF_GTC | sell-4    |
      | sellSideProvider  | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1  | ETH/FEB22 | buy  | 5      | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1  | ETH/FEB22 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1  | ETH/FEB22 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1            | ETH/FEB22 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider1 | ETH/FEB22 | sell | 2      | 15920  | 0                | TYPE_LIMIT | TIF_GTC | sell-2    |
      | sellSideProvider1 | ETH/FEB22 | sell | 1      | 15940  | 0                | TYPE_LIMIT | TIF_GTC | sell-3    |
      | sellSideProvider1 | ETH/FEB22 | sell | 3      | 17000  | 0                | TYPE_LIMIT | TIF_GTC | sell-4    |
      | sellSideProvider1 | ETH/FEB22 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the mark price should be "0" for the market "ETH/FEB23"

    # AC 0009-MRKP-024
    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    Then the mark price should be "15900" for the market "ETH/FEB22"

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15900      | TRADING_MODE_CONTINUOUS | 360     | 15801     | 15999     | 0            | 0              | 3             |
    Then the market data for the market "ETH/FEB22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15900      | TRADING_MODE_CONTINUOUS | 360     | 15801     | 15999     | 0            | 0              | 3             |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 2      | 15920 | 1                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1 | ETH/FEB22 | buy  | 2      | 15920 | 1                | TYPE_LIMIT | TIF_GTC |           |
    When the network moves ahead "1" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    Then the mark price should be "15900" for the market "ETH/FEB22"

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15940 | 1                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1 | ETH/FEB22 | buy  | 1      | 15940 | 1                | TYPE_LIMIT | TIF_GTC |           |
    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    Then the mark price should be "15900" for the market "ETH/FEB22"

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    Then the mark price should be "15900" for the market "ETH/FEB22"
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 17000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1 | ETH/FEB22 | buy  | 3      | 17000 | 0                | TYPE_LIMIT | TIF_GTC |           |
    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    Then the mark price should be "15900" for the market "ETH/FEB22"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB22"

    When the network moves ahead "9" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB22"
    Then the mark price should be "17000" for the market "ETH/FEB23"
    Then the mark price should be "17000" for the market "ETH/FEB22"

