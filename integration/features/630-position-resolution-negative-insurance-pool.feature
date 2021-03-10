Feature: Regression test for issue 630

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  BTC        | BTC   |  simple     | 0.2       | 0.1       | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 42               | 1                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Trader is being closed out.
# setup accounts
    Given the following traders:
      | name             | amount  |
      | sellSideProvider | 1000000 |
      | buySideProvider  | 1000000 |
      | traderGuy        | 240000  |
      | trader1          | 1000000 |
      | trader2          | 1000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | traderGuy        | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
      | trader1          | BTC   |
      | trader2          | BTC   |

    # Trigger an auction to set the mark price
    Then traders place following orders with references:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancels the following orders reference:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

# setup orderbook
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     |
      | sellSideProvider | ETH/DEC19 | sell | 200    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 200    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
    And All balances cumulated are worth "4240000"
    Then the margins levels for the traders are:
      | trader           | id        | maintenance | search | initial | release |
      | sellSideProvider | ETH/DEC19 | 2000        | 2200   | 2400    | 2800    |
    Then traders place following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     |
      | traderGuy | ETH/DEC19 | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader           | asset | id        | margin | general |
      | traderGuy        | BTC   | ETH/DEC19 | 0      | 0       |
      | sellSideProvider | BTC   | ETH/DEC19 | 240000 | 760000  |
    And the insurance pool balance is "240000" for the market "ETH/DEC19"
    And All balances cumulated are worth "4240000"
