Feature: Distressed parties should not have general balance left

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Upper bound breached
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount         |
      | party1    | ETH   | 10000000000000 |
      | party2    | ETH   | 10000000000000 |
      | party3    | ETH   | 24000          |
      | party4    | ETH   | 10000000000000 |
      | party5    | ETH   | 10000000000000 |
      | auxiliary | ETH   | 100000000000   |
      | aux2      | ETH   | 100000000000   |
      | lpprov    | ETH   | 10000000000000 |

    # Provide LP so market can leave opening auction
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 10000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC20 | 10000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | party1 | ETH/DEC20 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | party2 | ETH/DEC20 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    And the mark price should be "100" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then time is updated to "2020-10-16T00:01:00Z"

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC20 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party5 | ETH/DEC20 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-4     |
      | party3 | ETH/DEC20 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | party3 | ETH/DEC20 | sell | 40     | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general       |
      | party4 | ETH   | ETH/DEC20 | 360    | 9999999999640 |
      | party5 | ETH   | ETH/DEC20 | 372    | 9999999999528 |
    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp2 | party3 | ETH/DEC20 | 20000             | 0.1 | buy  | BID              | 10         | 10     | submission |
      | lp2 | party3 | ETH/DEC20 | 20000             | 0.1 | sell | ASK              | 10         | 10     | amendment  |
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp2 | party3 | ETH/DEC20 | 20000             | STATUS_ACTIVE |

    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        |
      | party3 | ETH/DEC20 | buy  | 189    | 100   | STATUS_ACTIVE |
      | party3 | ETH/DEC20 | sell | 117    | 130   | STATUS_ACTIVE |
    ## The sum of the margin + general account == 24000 - 10000 (commitment amount)
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party3 | ETH   | ETH/DEC20 | 2626   | 1374    |

    ## Now let's increase the mark price so party3 gets distressed
    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC20 | buy  | 40     | 165   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party1 | ETH/DEC20 | sell | 20     | 1850  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
    Then the mark price should be "120" for the market "ETH/DEC20"

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general       |
      | party3 | ETH   | ETH/DEC20 | 3152   | 923           |
      | party4 | ETH   | ETH/DEC20 | 160    | 9999999999640 |

    ## Now let's increase the mark price so party3 gets distressed
    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC20 | sell | 30     | 165   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party5 | ETH/DEC20 | buy  | 30     | 165   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the mark price should be "130" for the market "ETH/DEC20"

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party3 | ETH   | ETH/DEC20 | 4088   | 0       |
