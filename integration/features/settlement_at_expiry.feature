Feature: Test mark to market settlement

  Background:
    Given the markets start on "2019-11-30T00:00:00Z" and expire on "2019-12-31T23:59:59Z"
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Order cannot be placed once the market is expired
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount |
      | trader1  | ETH   | 10000  |
      | aux1     | ETH   | 100000 |
      | aux2     | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       |  999  | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux2     | ETH/DEC19 | sell | 1       | 1001  | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-3     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    # Set mark price
    Then the traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-5     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 1                | TYPE_LIMIT  | TIF_GTC | ref-6     |

    Then time is updated to "2020-01-01T01:01:01Z"
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
    Then the system should return error "OrderError: Invalid Market ID"

  Scenario: Settlement happened when market is being closed
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount |
      | trader1 | ETH   | 10000  |
      | trader2 | ETH   | 1000   |
      | trader3 | ETH   | 5000   |
      | aux1    | ETH   | 100000 |
      | aux2    | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       |  999  | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux2     | ETH/DEC19 | sell | 1       | 1001  | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-3     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    # Set mark price
    And the traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 1                | TYPE_LIMIT  | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | trader3 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-3     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | trader2 | ETH   | ETH/DEC19 | 132    | 868     |
      | trader3 | ETH   | ETH/DEC19 | 132    | 4868    |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "216000"

    # Close positions by aux traders
    When the traders place the following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux1    | ETH/DEC19 | sell | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1       | 1000  | 1                | TYPE_LIMIT  | TIF_GTC |

    Then time is updated to "2020-01-01T01:01:01Z"
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the system should return error "OrderError: Invalid Market ID"

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 8084    |
      | trader2 | ETH   | ETH/DEC19 | 0      | 1826    |
      | trader3 | ETH   | ETH/DEC19 | 0      | 5826    |
    And the cumulated balance for all accounts should be worth "214513"
