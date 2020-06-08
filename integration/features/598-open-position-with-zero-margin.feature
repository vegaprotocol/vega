Feature: Regression test for issue 598

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long |               tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | forward    |      0.001 | 0.00011407711613050422 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |
#     | ETH/DEC19 | BTC      | ETH       | ETH   |      1000 | simple     |       0.11 |                    0.1 |  0 |     0 |     0 |            1.4 |            1.2 |           1.1 |              42 |
#     | ETH/DEC19 | ETH      | BTC       | BTC   |         5 | forward    |       0.01 | 0.00011407711613050422 |  0 | 0.016 |  0.09 |            1.4 |            1.2 |           1.1 |              42 |

  Scenario: Open position but ZERO in margin account
    Given the following traders:
      | name   | amount |
      | edd    |   1000 |
      | barney |   1000 |
      | chris  |   1000 |
    Then I Expect the traders to have new general account:
      | name   | asset |
      | edd    | BTC   |
      | barney | BTC   |
      | chris  | BTC   |
    And "edd" general accounts balance is "1000"
    And "barney" general accounts balance is "1000"
    And "chris" general accounts balance is "1000"
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | edd    | ETH/DEC19 | sell |     10 |   101 |                0 | LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     12 |   102 |                0 | LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     13 |   103 |                0 | LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     14 |   104 |                0 | LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     15 |   105 |                0 | LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     10 |    99 |                0 | LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     12 |    98 |                0 | LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     13 |    97 |                0 | LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     14 |    96 |                0 | LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     15 |    95 |                0 | LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    571 |     429 |
      | barney | BTC   | ETH/DEC19 |    535 |     465 |
# next instruction will trade with edd
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | buy  |      10 |     0 |                1 | MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    571 |     429 |
      | chris  | BTC   | ETH/DEC19 |    109 |     891 |
# next instruction will trade with barney
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | sell |      10 |     0 |                1 | MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | chris  | BTC   | ETH/DEC19 |     0  |     980 |
      | barney | BTC   | ETH/DEC19 |    535 |     465 |
      | edd    | BTC   | ETH/DEC19 |    591 |     429 |
    Then the margins levels for the traders are:
      | trader | id        | maintenance | search | initial | release |
      | edd    | ETH/DEC19 |         502 |    552 |     602 |     702 |
      | barney | ETH/DEC19 |         451 |    496 |     541 |     631 |
      | chris  | ETH/DEC19 |           0 |      0 |       0 |       0 |
