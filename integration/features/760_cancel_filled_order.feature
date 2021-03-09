Feature: Close a filled order twice

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | base name | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | trading mode | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |

  Scenario: Traders place an order, a trade happens, and orders are cancelled after being filled
# setup accounts
    Given the following traders:
      | name             |    amount |
      | sellSideProvider | 100000000 |
      | buySideProvider  | 100000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
# setup orderbook
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |     10 |   120 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |     10 |   120 |                1 | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
    Then traders cancels the following filled orders reference:
      | trader           | reference       |
      | buySideProvider  | buy-provider-1  |
    Then traders cancels the following filled orders reference:
      | trader           | reference       |
      | buySideProvider  | buy-provider-1  |
    Then traders cancels the following filled orders reference:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
    And the insurance pool balance is "0" for the market "ETH/DEC19"
    And All balances cumulated are worth "200000000"

