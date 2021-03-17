Feature: Test crash on cancel of missing order

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: A non-existent party attempts to place an order
    Given missing traders place following orders with references:
      | trader        | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | missingTrader | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | missing-ref-1 |
    Then missing traders cancels the following orders reference:
      | trader        | reference     |
      | missingTrader | missing-ref-1 |
    Given missing traders place following orders with references:
      | trader        | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | missingTrader | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | missing-ref-2 |
    Then missing traders cancels the following orders reference:
      | trader        | reference     |
      | missingTrader | missing-ref-2 |
