Feature: test for issue 1920

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19 | ETH        | ETH   | 1000       | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              |

  Scenario: a trader place a new order in the system, margin are calculated, then the order is stopped, the margin is released
    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    And "trader1" general accounts balance is "10000"
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |      1 |  1000 |                0 | TYPE_LIMIT | TIF_FOK |
    Then the margins levels for the traders are:
      | trader  | id        | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         100 |    110 |     120 |     140 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |      0 |   10000 |
