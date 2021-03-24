Feature: Test mark to market settlement

  Background:
    Given the markets start on "2019-11-30T00:00:00Z" and expire on "2019-12-31T23:59:59Z"
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Order cannot be placed once the market is expired
    Given the traders make the following deposits on asset's general account:
      | trader   | asset | amount |
      | trader1  | ETH   | 10000  |
      | aux1     | ETH   | 100000 |
      | aux2     | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       |  999  | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux2     | ETH/DEC19 | sell | 1       | 1001  | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-3     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-4     |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "1000"

    # Set mark price
    Then traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-5     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 1                | TYPE_LIMIT  | TIF_GTC | ref-6     |

    Then time is updated to "2020-01-01T01:01:01Z"
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
    Then the system should return error "OrderError: Invalid Market ID"

  Scenario: Settlement happened when market is being closed
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 10000  |
      | trader2 | ETH   | 1000   |
      | trader3 | ETH   | 5000   |
      | aux1    | ETH   | 100000 |
      | aux2    | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       |  999  | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux2     | ETH/DEC19 | sell | 1       | 1001  | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-3     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-4     |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "1000"

    # Set mark price
    And traders place the following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     | reference |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 1                | TYPE_LIMIT  | TIF_GTC | ref-2     |

    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | trader3 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-3     |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | trader2 | ETH   | ETH/DEC19 | 132    | 868     |
      | trader3 | ETH   | ETH/DEC19 | 132    | 4868    |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    And Cumulated balance for all accounts is worth "216000"

    # Close positions by aux traders
    When traders place the following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux1    | ETH/DEC19 | sell | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1       | 1000  | 1                | TYPE_LIMIT  | TIF_GTC |

    Then time is updated to "2020-01-01T01:01:01Z"
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the system should return error "OrderError: Invalid Market ID"

    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 8084    |
      | trader2 | ETH   | ETH/DEC19 | 0      | 1826    |
      | trader3 | ETH   | ETH/DEC19 | 0      | 5826    |
    And Cumulated balance for all accounts is worth "214513"
