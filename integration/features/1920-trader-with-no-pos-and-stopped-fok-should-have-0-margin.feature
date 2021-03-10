Feature: test for issue 1920

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  ETH        | ETH   |  simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 1                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader place a new order in the system, margin are calculated, then the order is stopped, the margin is released
    Given the following traders:
      | name    | amount    |
      | trader1 | 10000     |
      | trader2 | 100000000 |
      | trader3 | 100000000 |
      | trader4 | 100000000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
      | trader4 | ETH   |
    And "trader1" general accounts balance is "10000"
    Then traders place following orders:
      | trader  | id        | side | volume | price | resulting trades | type       | tif     |
      | trader2 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | trader3 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | trader4 | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC19 | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "1000"

    Then traders place following orders:
      | trader  | id        | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_FOK |

    Then the margins levels for the traders are:
      | trader  | id        | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 10000   |
