Feature: Test LP orders

  Scenario: 001, create liquidity provisions (0038-OLIQ-additional-tests)
  Background:

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party1 | ETH/DEC19 | 50000             | 0.1 | buy  | BID              | 500        | 10     | submission |
      | lp1 | party1 | ETH/DEC19 | 50000             | 0.1 | sell | ASK              | 500        | 10     | submission |
    And the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 1      | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 120   | 1      |
      | buy  | 80    | 1      |
      | buy  | 100   | 1      |
      | sell | 100   | 1      |

    Then the opening auction period ends for market "ETH/DEC19"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the following trades should be executed:
      | buyer | price | size | seller    |
      | aux2  | 100   | 1    | auxiliary |
    And the mark price should be "100" for the market "ETH/DEC19"

    Then the orders should have the following states:
      | party     | market id | side | volume | price | status        |
      | auxiliary | ETH/DEC19 | buy  | 1      | 80    | STATUS_ACTIVE |
      | auxiliary | ETH/DEC19 | sell | 1      | 120   | STATUS_ACTIVE |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | STATUS_ACTIVE |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | STATUS_ACTIVE |

    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | party1           | ETH/DEC19 | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-1        |
      | party1           | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-2        |
    Then the orders should have the following states:
      | party            | market id | side | volume | price | status        |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | STATUS_ACTIVE |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | STATUS_ACTIVE |
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | party1 | ETH/DEC19 | 50000             | STATUS_ACTIVE |
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        |
      | party1 | ETH/DEC19 | buy  | 499    | 100   | STATUS_ACTIVE |
      | party1 | ETH/DEC19 | sell | 384    | 130   | STATUS_ACTIVE |

  Scenario: 002, create liquidity provisions (0038-OLIQ-additional-tests); test decimal; asset 3; market 1; position:2 AC: 0070-MKTD-004;0070-MKTD-005; 0070-MKTD-006; 0070-MKTD-007;0070-MKTD-008
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 3              |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1              | 2                       | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount          |
      | party1           | ETH   | 100000000000    |
      | sellSideProvider | ETH   | 100000000000    |
      | buySideProvider  | ETH   | 100000000000    |
      | auxiliary        | ETH   | 100000000000000 |
      | aux2             | ETH   | 100000000000000 |
    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 100    | 800   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 100    | 1200  | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party1 | ETH/DEC19 | 50000000          | 0.1 | buy  | BID              | 500        | 100    | submission |
      | lp1 | party1 | ETH/DEC19 | 50000000          | 0.1 | sell | ASK              | 500        | 100    | submission |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1200  | 100    |
      | buy  | 800   | 100    |
      | buy  | 1000  | 100    |
      | sell | 1000  | 100    |

    Then the opening auction period ends for market "ETH/DEC19"

    Then the following trades should be executed:
      | buyer | price | size | seller    |
      | aux2  | 1000  | 100  | auxiliary |
    And the mark price should be "1000" for the market "ETH/DEC19"
    Then the orders should have the following states:
      | party     | market id | side | volume | price | status        |
      | auxiliary | ETH/DEC19 | buy  | 100    | 800   | STATUS_ACTIVE |
      | auxiliary | ETH/DEC19 | sell | 100    | 1200  | STATUS_ACTIVE |
      | aux2      | ETH/DEC19 | buy  | 100    | 1000  | STATUS_ACTIVE |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 100000 | 1200  | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 100000 | 800   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | party1           | ETH/DEC19 | buy  | 5000   | 1100  | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-1        |
      | party1           | ETH/DEC19 | sell | 5000   | 1200  | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-2        |
    Then the orders should have the following states:
      | party            | market id | side | volume | price | status        |
      | sellSideProvider | ETH/DEC19 | sell | 100000 | 1200  | STATUS_ACTIVE |
      | buySideProvider  | ETH/DEC19 | buy  | 100000 | 800   | STATUS_ACTIVE |
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | party1 | ETH/DEC19 | 50000000          | STATUS_ACTIVE |
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        |
      | party1 | ETH/DEC19 | buy  | 44500  | 1000  | STATUS_ACTIVE |
      | party1 | ETH/DEC19 | sell | 33847  | 1300  | STATUS_ACTIVE |
