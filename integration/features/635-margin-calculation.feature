Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  BTC        | BTC   |  simple     | 0.2       | 0.1       | 0              | 0.016           | 2.0   | 5              | 4              | 3.2           | 42               | 0                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Traded out position but monies left in margin account
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount     |
      | traderGuy        | BTC   | 1000000000 |
      | sellSideProvider | BTC   | 1000000000 |
      | buySideProvider  | BTC   | 1000000000 |
# setup previous mark price
    Then traders place following orders:
      | trader           | market id | side | volume | price    | resulting trades | type       | tif     |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC |
# setup orderbook
    Then traders place following orders:
      | trader           | market id | side | volume | price    | resulting trades | type       | tif     |
      | sellSideProvider | ETH/DEC19 | sell | 100    | 25000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/DEC19 | sell | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 3      | 9600000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 15     | 9000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 8700000  | 0                | TYPE_LIMIT | TIF_GTC |
# buy 13@150
    Then traders place following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC |
# checking margins
    Then traders have the following account balances:
      | trader    | asset | market id | margin    | general   |
      | traderGuy | BTC   | ETH/DEC19 | 394400032 | 611199968 |
# checking margins levels
    Then the margins levels for the traders are:
      | trader    | market id | maintenance | search    | initial   | release   |
      | traderGuy | ETH/DEC19 | 98600008    | 315520025 | 394400032 | 493000040 |
