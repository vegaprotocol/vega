Feature: Test crash on cancel of missing order

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19 | BTC        | BTC   | 100        | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              |

  Scenario: A non-existent party attempts to place an order
    Given missing traders place following orders with references:
      | trader        | id        | type | volume | price | resulting trades | type  | tif | reference     |
      | missingTrader | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | missing-ref-1 |
    Then missing traders cancels the following orders reference:
      | trader        | reference     |
      | missingTrader | missing-ref-1 |
    Given missing traders place following orders with references:
      | trader        | id        | type | volume | price | resulting trades | type  | tif | reference     |
      | missingTrader | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | missing-ref-2 |
    Then missing traders cancels the following orders reference:
      | trader        | reference     |
      | missingTrader | missing-ref-2 |
