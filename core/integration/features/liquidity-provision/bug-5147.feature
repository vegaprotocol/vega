Feature: Test LP orders with different decimals for market and asset

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00011407711613050422 | 0  | 0.016 | 1.5   |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees         | price monitoring | data source config     | position decimal places | decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 5                       | 5              | 5e-2                   | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: create liquidity provisions
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount                      |
      | party1           | ETH   | 100000000000000000000000000 |
      | party2           | ETH   | 100000000000000000000000000 |
      | party3           | ETH   | 100000000000000000000000000 |
      | sellSideProvider | ETH   | 100000000000000000000000000 |
      | buySideProvider  | ETH   | 100000000000000000000000000 |
      | auxiliary        | ETH   | 100000000000000000000000000 |
      | aux2             | ETH   | 100000000000000000000000000 |
      | lpprov           | ETH   | 100000000000000000000000000 |

    When the parties place the following orders:
      | party     | market id | side | volume | price     | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 100    | 80000000  | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 100    | 120000000 | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 100    | 100000000 | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 100    | 100000000 | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount  | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 552900000000000000 | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 552900000000000000 | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price       | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 500000 | 100100000   | 0                | TYPE_LIMIT | TIF_GTC | b1        |
      | party3 | ETH/DEC19 | sell | 500000 | 95100000    | 1                | TYPE_LIMIT | TIF_GTC | s1        |
      | party2 | ETH/DEC19 | buy  | 500000 | 90000000    | 0                | TYPE_LIMIT | TIF_GTC | b2        |
      | party3 | ETH/DEC19 | sell | 500000 | 120000000   | 0                | TYPE_LIMIT | TIF_GTC | s2        |
      | party2 | ETH/DEC19 | buy  | 100000 | 10000000    | 0                | TYPE_LIMIT | TIF_GTC | b3        |
      | party3 | ETH/DEC19 | sell | 100000 | 10000000000 | 0                | TYPE_LIMIT | TIF_GTC | s3        |
      | party2 | ETH/DEC19 | sell | 100000 | 95100000    | 0                | TYPE_LIMIT | TIF_GTC | b4        |
      | party3 | ETH/DEC19 | buy  | 100000 | 100100000   | 1                | TYPE_LIMIT | TIF_GTC | s4        |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount         | fee | side | pegged reference | proportion | offset | lp type    |
      | lp2 | party1 | ETH/DEC19 | 3905000000000000000000000 | 0.3 | buy  | BID              | 2          | 100000 | submission |
      | lp2 | party1 | ETH/DEC19 | 3905000000000000000000000 | 0.3 | sell | ASK              | 13         | 100000 | amendment  |

    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount         | status        |
      | lp2 | party1 | ETH/DEC19 | 3905000000000000000000000 | STATUS_ACTIVE |

    Then the orders should have the following states:
      | party  | market id | side | volume    | price     | status        | reference |
      | party1 | ETH/DEC19 | buy  | 434371524 | 89900000  | STATUS_ACTIVE | lp2       |
      | party1 | ETH/DEC19 | sell | 325145712 | 120100000 | STATUS_ACTIVE | lp2       |
