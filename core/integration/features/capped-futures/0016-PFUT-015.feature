Feature: Pegged orders are capped to max price.

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | DAI | 5              |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 1.0              | 20s         | 10             |  
    And the log normal risk model named "dai-lognormal-risk":
      | risk aversion | tau         | mu | r | sigma |
      | 0.00001       | 0.000114077 | 0  | 0 | 0.41  |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model         | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor | sla params      | max price cap | binary | fully collateralised |
      | DAI/DEC22 | DAI        | DAI   | lqm-params           | dai-lognormal-risk | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 5              | 0.25                   | 0                         | default-futures | 4500000000    | false  | false                |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |

  @MidPrice @NoPerp @Capped
  Scenario: 0016-PFUT-015: pegged orders are capped to max price.
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |
      | party3 | DAI   | 110000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | lp-1      | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party1 | DAI/DEC22 | 5         | 3                    | buy  | MID              | 5      | 10     |
      | party1 | DAI/DEC22 | 5         | 3                    | sell | MID              | 5      | 10     |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party3 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party3 | DAI/DEC22 | sell | 1      | 4499999999 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party3 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 2649999999       | 4499999999              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4499999999 | 1      |
      | sell | 2650000009 | 5      |
      | buy  | 2649999990 | 5      |
      | buy  | 800000000  | 1      |
    # Ensure the price cap is enforced on all orders
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference | error               |
      | party3 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-3  | invalid order price |

    # Now move mid price close to the max price
    When the parties cancel the following orders:
      | party  | reference |
      | party2 | party2-1  |
    And the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 4499999998 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 4499999998            | 4499999998       | 4499999999              |
    # Now the sell order should be capped to max price, buy order is offset by 10
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4499999999 | 6      |
      | buy  | 4499999998 | 1      |
      | buy  | 4499999989 | 5      |
