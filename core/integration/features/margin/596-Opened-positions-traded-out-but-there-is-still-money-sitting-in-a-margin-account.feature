Feature: Regression test for issue 596

  Background:
    Given the markets:
      | id        | quote name | asset | auction duration | risk model                    | margin calculator         | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | 1                | default-log-normal-risk-model | default-margin-calculator | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Traded out position but monies left in margin account
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | edd    | BTC   | 10000   |
      | barney | BTC   | 10000   |
      | chris  | BTC   | 10000   |
      | party1 | BTC   | 1000000 |
      | party2 | BTC   | 1000000 |
      | aux    | BTC   | 1000    |
      | lpprov | BTC   | 1000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | edd    | ETH/DEC19 | sell | 20     | 101   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | edd    | ETH/DEC19 | sell | 20     | 102   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | edd    | ETH/DEC19 | sell | 10     | 103   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | edd    | ETH/DEC19 | sell | 15     | 104   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | edd    | ETH/DEC19 | sell | 30     | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | barney | ETH/DEC19 | buy  | 20     | 99    | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | barney | ETH/DEC19 | buy  | 14     | 97    | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |
      | barney | ETH/DEC19 | buy  | 20     | 96    | 0                | TYPE_LIMIT | TIF_GTC | ref-9     |
      | barney | ETH/DEC19 | buy  | 5      | 95    | 0                | TYPE_LIMIT | TIF_GTC | ref-10    |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 848    | 9152    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | chris | ETH/DEC19 | buy  | 50     | 110   | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1413   | 8527    |
      | chris  | BTC   | ETH/DEC19 | 790    | 8761    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    And the cumulated balance for all accounts should be worth "3031000"
    #then chris is trading out
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | chris | ETH/DEC19 | sell | 50     | 90    | 4                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1763   | 8527    |
      | chris  | BTC   | ETH/DEC19 | 0      | 8808    |
      | barney | BTC   | ETH/DEC19 | 3810   | 6092    |
    And the cumulated balance for all accounts should be worth "3031000"

  Scenario: Traded out position, with cancelled half traded order, but monies left in margin account
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | edd    | BTC   | 10000   |
      | barney | BTC   | 10000   |
      | chris  | BTC   | 10000   |
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
      | edd    | ETH/DEC19 | sell | 20     | 101   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | edd    | ETH/DEC19 | sell | 20     | 102   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | edd    | ETH/DEC19 | sell | 10     | 103   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | edd    | ETH/DEC19 | sell | 15     | 104   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | edd    | ETH/DEC19 | sell | 30     | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | barney | ETH/DEC19 | buy  | 20     | 99    | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | barney | ETH/DEC19 | buy  | 14     | 97    | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |
      | barney | ETH/DEC19 | buy  | 20     | 96    | 0                | TYPE_LIMIT | TIF_GTC | ref-10    |
      | barney | ETH/DEC19 | buy  | 5      | 95    | 0                | TYPE_LIMIT | TIF_GTC | ref-11    |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 848    | 9152    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    # Chris place an order for a volume of 60, but only 2 trades happen at that price
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference            |
      | chris | ETH/DEC19 | buy  | 60     | 102   | 2                | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 961    | 9019    |
      | chris  | BTC   | ETH/DEC19 | 607    | 9007    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    And the cumulated balance for all accounts should be worth "3031000"
    Then the parties cancel the following orders:
      | party | reference            |
      | chris | chris-id-1-to-cancel |
    # then chris is trading out
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | chris | ETH/DEC19 | sell | 40     | 90    | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1161   | 9019    |
      | chris  | BTC   | ETH/DEC19 | 0      | 9072    |
      | barney | BTC   | ETH/DEC19 | 1536   | 8412    |
    And the cumulated balance for all accounts should be worth "3031000"
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | barney | ETH/DEC19 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 921    | 9019    |
      | chris  | BTC   | ETH/DEC19 | 0      | 9072    |
      | barney | BTC   | ETH/DEC19 | 1998   | 8179    |
    And the cumulated balance for all accounts should be worth "3031000"
