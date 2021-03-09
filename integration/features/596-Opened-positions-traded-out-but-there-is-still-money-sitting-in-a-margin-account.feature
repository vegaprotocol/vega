Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19 | BTC        | BTC   | 100        | forward    | 0.001     | 0.00011407711613050422 | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              |
      
  Scenario: Traded out position but monies left in margin account
    Given the following traders:
      | name   | amount |
      | edd    |   10000 |
      | barney |   10000 |
      | chris  |   10000 |
    Then I Expect the traders to have new general account:
      | name   | asset |
      | edd    | BTC   |
      | barney | BTC   |
      | chris  | BTC   |
    And "edd" general accounts balance is "10000"
    And "barney" general accounts balance is "10000"
    And "chris" general accounts balance is "10000"
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | edd    | ETH/DEC19 | sell |     20 |   101 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     20 |   102 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     10 |   103 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     15 |   104 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     30 |   105 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     20 |    99 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     12 |    98 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     14 |    97 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     20 |    96 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     5  |    95 |                0 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    848 |    9152 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | chris  | ETH/DEC19 | buy  |     50 |   110 |                3 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    933 |    9007 |
      | chris  | BTC   | ETH/DEC19 |    790 |    9270 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    And All balances cumulated are worth "30000"
# then chris is trading out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | chris  | ETH/DEC19 | sell |     50 |    90 |                4 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |   1283 |    9007 |
      | chris  | BTC   | ETH/DEC19 |      0 |    9808 |
      | barney | BTC   | ETH/DEC19 |    630 |    9272 |
    And All balances cumulated are worth "30000"


  Scenario: Traded out position, with cancelled half traded order, but monies left in margin account
    Given the following traders:
      | name   | amount |
      | edd    |   10000 |
      | barney |   10000 |
      | chris  |   10000 |
    Then I Expect the traders to have new general account:
      | name   | asset |
      | edd    | BTC   |
      | barney | BTC   |
      | chris  | BTC   |
    And "edd" general accounts balance is "10000"
    And "barney" general accounts balance is "10000"
    And "chris" general accounts balance is "10000"
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | edd    | ETH/DEC19 | sell |     20 |   101 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     20 |   102 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     10 |   103 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     15 |   104 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     30 |   105 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     20 |    99 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     12 |    98 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     14 |    97 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     20 |    96 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     5  |    95 |                0 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    848 |    9152 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
# Chris place an order for a volume of 60, but only 2 trades happen at that price
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference            |
      | chris  | ETH/DEC19 | buy  |     60 |   102 |                2 | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    961 |    9019 |
      | chris  | BTC   | ETH/DEC19 |    607 |    9413 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    And All balances cumulated are worth "30000"
    Then traders cancels the following orders reference:
      | trader | reference            |
      | chris  | chris-id-1-to-cancel |
# then chris is trading out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | chris  | ETH/DEC19 | sell |     40 |    90 |                3 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |   1161 |    9019 |
      | chris  | BTC   | ETH/DEC19 |      0 |    9872 |
      | barney | BTC   | ETH/DEC19 |    624 |    9324 |
    And All balances cumulated are worth "30000"
   Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | barney | ETH/DEC19 | buy  |      1 |   105 |                1 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    921 |    9019 |
      | chris  | BTC   | ETH/DEC19 |      0 |    9872 |
      | barney | BTC   | ETH/DEC19 |    964 |    9224 |
    And All balances cumulated are worth "30000"
