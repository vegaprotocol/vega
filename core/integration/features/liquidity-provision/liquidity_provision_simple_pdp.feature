Feature: Test LP orders

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2                       | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: create liquidity provisions
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |
      | lpprov           | ETH   | 100000000 |

    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 100    | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 100    | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 100000 | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 100000 | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | party1           | ETH/DEC19 | buy  | 500    | 110   | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-1        |
      | party1           | ETH/DEC19 | sell | 500    | 120   | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-2        |

    Then the market data for the market "ETH/DEC19" should be:
      | static mid price | best static bid price | best static offer price |
      | 115              | 110                   | 120                     |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp2 | party1 | ETH/DEC19 | 50000             | 0.1 | buy  | BID              | 500        | 10     | submission |
      | lp2 | party1 | ETH/DEC19 | 50000             | 0.1 | sell | ASK              | 500        | 10     | submission |
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp2 | party1 | ETH/DEC19 | 50000             | STATUS_ACTIVE |

    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 49450  | 100   | STATUS_ACTIVE | lp2       |
      | party1 | ETH/DEC19 | sell | 38000  | 130   | STATUS_ACTIVE | lp2       |
