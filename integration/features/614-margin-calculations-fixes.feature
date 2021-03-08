Feature: test bugfix 614 for margin calculations

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | base name | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | trading mode | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC       | ETH        | ETH   | 94         | simple     | 0.2       | 0.1       | 0              | 0               | 0     | 5              | 4              | 3.2           | 100              | 0                | continuous   | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: CASE-1: Trader submits long order that will trade - new formula & high exit price
    Given the following traders:
      | name    | amount |
      | chris   |  10000 |
      | edd     |  10000 |
      | barney  |  10000 |
      | rebecca |  10000 |
      | tamlyn  |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | chris   |   ETH |
      | edd     |   ETH |
      | barney  |   ETH |
      | rebecca |   ETH |
      | tamlyn  |   ETH |
    And "chris" general accounts balance is "10000"
    And "edd" general accounts balance is "10000"
    And "barney" general accounts balance is "10000"
    And "rebecca" general accounts balance is "10000"
    And "tamlyn" general accounts balance is "10000"
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | chris   | ETH/DEC19 | sell |    100 |   250 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd     | ETH/DEC19 | sell |     11 |   140 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney  | ETH/DEC19 | sell |      2 |   112 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney  | ETH/DEC19 | buy  |      1 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd     | ETH/DEC19 | buy  |      3 |    96 |                0 | TYPE_LIMIT | TIF_GTC |
      | chris   | ETH/DEC19 | buy  |     15 |    90 |                0 | TYPE_LIMIT | TIF_GTC |
      | rebecca | ETH/DEC19 | buy  |     50 |    87 |                0 | TYPE_LIMIT | TIF_GTC |
      # this is now the actual trader that we are testing
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | tamlyn | ETH/DEC19 | buy  |     13 |   150 |                2 | TYPE_LIMIT | TIF_GTC |
    Then the margins levels for the traders are:
      | trader | id        | maintenance | search | initial | release |
      | tamlyn | ETH/DEC19 |         988 |   3161 |    3952 |    4940 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | tamlyn  | ETH   | ETH/DEC19 |   3952 |    6104 |
      | chris   | ETH   | ETH/DEC19 |   3760 |    6240 |
      | edd     | ETH   | ETH/DEC19 |   5456 |    4544 |
      | barney  | ETH   | ETH/DEC19 |    992 |    8952 |
      | rebecca | ETH   | ETH/DEC19 |   3760 |    6240 |
    And All balances cumulated are worth "50000"
