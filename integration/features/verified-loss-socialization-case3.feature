Feature: Test loss socialization case 3

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: case 3 from https://docs.google.com/spreadsheets/d/1CIPH0aQmIKj6YeFW9ApP_l-jwB4OcsNQ/edit#gid=1555964910
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | trader1          | BTC   | 2000      |
      | trader2          | BTC   | 10000     |
      | trader3          | BTC   | 3000      |
      | trader4          | BTC   | 10000     |
      | trader5          | BTC   | 100000000 |
      | trader6          | BTC   | 100000000 |

# setup order book
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | trader5          | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GFA | buy-provider-t5 |
      | trader6          | ETH/DEC19 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GFA | buy-provider-t6 |

    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"

# trade 1 occur
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 30     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 30     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
# trade 2 occur
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | sell | 60     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 60     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
# trade 3 occur
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader4 | ETH/DEC19 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# order book volume change
    Then traders cancel the following orders:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 300   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

# trade 4 occur
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader4 | ETH/DEC19 | sell | 10     | 180   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# check positions
    Then traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | -2000        |
      | trader2 | 100    | 7200           | -2455        |
      | trader3 | 0      | 0              | -3000        |
      | trader4 | 0      | 0              | 528          |
    And the insurance pool balance is "0" for the market "ETH/DEC19"
