Feature: Close a filled order twice

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Traders place an order, a trade happens, and orders are cancelled after being filled
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
# setup orderbook
    Then traders place following orders with references:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 10     | 120   | 1                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
    Then traders attempt to cancel the following filled orders:
      | trader          | reference      |
      | buySideProvider | buy-provider-1 |
    Then traders attempt to cancel the following filled orders:
      | trader          | reference      |
      | buySideProvider | buy-provider-1 |
    Then traders attempt to cancel the following filled orders:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
    And the insurance pool balance is "0" for the market "ETH/DEC19"
    And Cumulated balance for all accounts is worth "200000000"

