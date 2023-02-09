Feature: Regression test for issue 598

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Open position but ZERO in margin account
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | edd    | BTC   | 1000    |
      | barney | BTC   | 1000    |
      | chris  | BTC   | 1000    |
      | party1 | BTC   | 1000000 |
      | party2 | BTC   | 1000000 |
      | aux    | BTC   | 1000    |
      | lpprov | BTC   | 1000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | edd    | ETH/DEC19 | sell | 10     | 101   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | edd    | ETH/DEC19 | sell | 12     | 102   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | edd    | ETH/DEC19 | sell | 13     | 103   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | edd    | ETH/DEC19 | sell | 14     | 104   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | edd    | ETH/DEC19 | sell | 15     | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | barney | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | barney | ETH/DEC19 | buy  | 13     | 97    | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |
      | barney | ETH/DEC19 | buy  | 14     | 96    | 0                | TYPE_LIMIT | TIF_GTC | ref-9     |
      | barney | ETH/DEC19 | buy  | 15     | 95    | 0                | TYPE_LIMIT | TIF_GTC | ref-10    |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 571    | 429     |
      | barney | BTC   | ETH/DEC19 | 535    | 465     |
    # next instruction will trade with edd
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris | ETH/DEC19 | buy  | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | edd   | BTC   | ETH/DEC19 | 571    | 429     |
      | chris | BTC   | ETH/DEC19 | 109    | 790     |
    # next instruction will trade with barney
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris | ETH/DEC19 | sell | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | chris  | BTC   | ETH/DEC19 | 0      | 780     |
      | barney | BTC   | ETH/DEC19 | 535    | 465     |
      | edd    | BTC   | ETH/DEC19 | 591    | 429     |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | edd    | ETH/DEC19 | 502         | 552    | 602     | 702     |
      | barney | ETH/DEC19 | 451         | 496    | 541     | 631     |
      | chris  | ETH/DEC19 | 0           | 0      | 0       | 0       |
