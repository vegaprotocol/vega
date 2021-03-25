Feature: Test mark to market settlement

  Background:
    Given the markets start on "2019-11-30T00:00:00Z" and expire on "2019-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | ETH        | ETH   | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
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
    Then traders place following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux1     | ETH/DEC19 | buy  | 1       |  999  | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2     | ETH/DEC19 | sell | 1       | 1001  | 0                | TYPE_LIMIT  | TIF_GTC |

    # Set mark price
    Then traders place following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0               | TYPE_LIMIT  | TIF_GTC |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 1               | TYPE_LIMIT  | TIF_GTC |

    Then time is updated to "2020-01-01T01:01:01Z"
    When traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
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
    Then traders place following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux1     | ETH/DEC19 | buy  | 1       |  999  | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2     | ETH/DEC19 | sell | 1       | 1001  | 0                | TYPE_LIMIT  | TIF_GTC |

    # Set mark price
    Then traders place following orders:
      | trader   | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux1     | ETH/DEC19 | buy  | 1       | 1000  | 0               | TYPE_LIMIT  | TIF_GTC |
      | aux2     | ETH/DEC19 | sell | 1       | 1000  | 1               | TYPE_LIMIT  | TIF_GTC |

    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | trader2 | ETH   | ETH/DEC19 | 132    | 868     |
      | trader3 | ETH   | ETH/DEC19 | 132    | 4868    |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    And Cumulated balance for all accounts is worth "216000"

    # Close positions by aux traders
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux1    | ETH/DEC19 | sell | 1       | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1       | 1000  | 1                | TYPE_LIMIT  | TIF_GTC |

    Then time is updated to "2020-01-01T01:01:01Z"
    When traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the system should return error "OrderError: Invalid Market ID"

    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 8084    |
      | trader2 | ETH   | ETH/DEC19 | 0      | 2784    |
      | trader3 | ETH   | ETH/DEC19 | 0      | 4868    |
    And Cumulated balance for all accounts is worth "215736"
