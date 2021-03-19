Feature: Regression test for issue 630

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0.2       | 0.1       | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Trader is being closed out.
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount  |
      | sellSideProvider | BTC   | 1000000 |
      | buySideProvider  | BTC   | 1000000 |
      | traderGuy        | BTC   | 240000  |
      | trader1          | BTC   | 1000000 |
      | trader2          | BTC   | 1000000 |
      | aux              | BTC   | 100000  |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"

# setup orderbook
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 200    | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 200    | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And Cumulated balance for all accounts is worth "4340000"
    Then the margins levels for the traders are:
      | trader           | market id | maintenance | search | initial | release |
      | sellSideProvider | ETH/DEC19 | 2000        | 2200   | 2400    | 2800    |
    Then traders place following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then traders have the following account balances:
      | trader           | asset | market id | margin | general |
      | traderGuy        | BTC   | ETH/DEC19 | 0      | 0       |
      | sellSideProvider | BTC   | ETH/DEC19 | 240000 | 760000  |
    And the insurance pool balance is "240000" for the market "ETH/DEC19"
    And Cumulated balance for all accounts is worth "4340000"
