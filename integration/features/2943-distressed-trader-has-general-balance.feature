Feature: Distressed traders should not have general balance left

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the markets:
      | id        | quote name | asset | maturity date        | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: Upper bound breached
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount         |
      | trader1   | ETH   | 10000000000000 |
      | trader2   | ETH   | 10000000000000 |
      | trader3   | ETH   | 24000          |
      | trader4   | ETH   | 10000000000000 |
      | trader5   | ETH   | 10000000000000 |
      | auxiliary | ETH   | 100000000000   |
      | aux2      | ETH   | 100000000000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | trader1 | ETH/DEC20 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | trader2 | ETH/DEC20 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    And the mark price should be "100" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then time is updated to "2020-10-16T00:01:00Z"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC20 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/DEC20 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-1     |
      | trader3 | ETH/DEC20 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | trader3 | ETH/DEC20 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general       |
      | trader4 | ETH   | ETH/DEC20 | 360    | 9999999999640 |
      | trader5 | ETH   | ETH/DEC20 | 372    | 9999999999628 |
    And clear order events
    Then the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset |
      | lp1 | trader3 | ETH/DEC20 | 10000             | 0.1 | buy  | BID              | 10         | -10    |
      | lp1 | trader3 | ETH/DEC20 | 10000             | 0.1 | sell | ASK              | 10         | 10     |
    Then the liquidity provisions should have the following states:
      | id  | party   | market    | commitment amount | status        |
      | lp1 | trader3 | ETH/DEC20 | 10000             | STATUS_ACTIVE |

    Then the orders should have the following states:
      | trader  | market id | side | volume | price | status        |
      | trader3 | ETH/DEC20 | buy  | 945    | 100   | STATUS_ACTIVE |
      | trader3 | ETH/DEC20 | sell | 724    | 130   | STATUS_ACTIVE |
    ## The sum of the margin + general account == 10000 - 10000 (commitment amount)
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 12606  | 1394    |

    ## Now let's increase the mark price so trader3 gets distressed
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/DEC20 | buy  | 20     | 165   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the mark price should be "120" for the market "ETH/DEC20"

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 15127  | 0       |

    ## Now let's increase the mark price so trader3 gets distressed
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC20 | sell | 30     | 165   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader5 | ETH/DEC20 | buy  | 30     | 165   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And the mark price should be "130" for the market "ETH/DEC20"

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 16388  | 0       |
