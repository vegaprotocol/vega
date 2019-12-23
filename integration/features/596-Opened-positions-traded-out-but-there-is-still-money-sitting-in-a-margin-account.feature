Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short |               tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | forward    |      0.001 | 0.00011407711613050422 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |

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
      | edd    | ETH/DEC19 | sell |     20 |   101 |                0 | LIMIT | GTC |
      | edd    | ETH/DEC19 | sell |     20 |   102 |                0 | LIMIT | GTC |
      | edd    | ETH/DEC19 | sell |     10 |   103 |                0 | LIMIT | GTC |
      | edd    | ETH/DEC19 | sell |     15 |   104 |                0 | LIMIT | GTC |
      | edd    | ETH/DEC19 | sell |     30 |   105 |                0 | LIMIT | GTC |
      | barney | ETH/DEC19 | buy  |     20 |    99 |                0 | LIMIT | GTC |
      | barney | ETH/DEC19 | buy  |     12 |    98 |                0 | LIMIT | GTC |
      | barney | ETH/DEC19 | buy  |     14 |    97 |                0 | LIMIT | GTC |
      | barney | ETH/DEC19 | buy  |     20 |    96 |                0 | LIMIT | GTC |
      | barney | ETH/DEC19 | buy  |     5  |    95 |                0 | LIMIT | GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    848 |    9152 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | chris  | ETH/DEC19 | buy  |     50 |   110 |                3 | LIMIT | GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    933 |    9007 |
      | chris  | BTC   | ETH/DEC19 |     12 |   10048 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    And All balances cumulated are worth "30000"
# then cris is trading out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | chris  | ETH/DEC19 | sell |     50 |    90 |                4 | LIMIT | GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |   1283 |    9007 |
      | chris  | BTC   | ETH/DEC19 |      0 |    9808 |
      | barney | BTC   | ETH/DEC19 |    496 |    9406 |
    And All balances cumulated are worth "30000"
