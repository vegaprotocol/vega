Feature: Regression test for issue 630

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |  10300000 | simple     |        0.2 |      0.1 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |

  Scenario: Trader is being closed out.
# setup accounts
    Given the following traders:
      | name             |      amount |
      | traderGuy        |  1000000000 |
      | sellSideProvider | 10000000000 |
      | buySideProvider | 10000000000 |
# setup previous mark price
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |      1 | 10300000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 | 10300000 |                1 | LIMIT | GTC |
# setup order book
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |    100 | 25000000 |                0 | LIMIT | GTC |
      | sellSideProvider | ETH/DEC19 | sell |     11 | 14000000 |                0 | LIMIT | GTC |
      | sellSideProvider | ETH/DEC19 | sell |      2 | 11200000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 | 10000000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      3 |  9600000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |     15 |  9000000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |     50 |  8700000 |                0 | LIMIT | GTC |
# buy 13@150
    Then traders place following orders:
      | trader    | id        | type | volume |    price | resulting trades | type  | tif |
      | traderGuy | ETH/DEC19 | buy  |     13 | 15000000 |                2 | LIMIT | GTC |
# checking margins
    Then the margins levels for the traders are:
      | trader    | id        | maintenance |    search |   initial |   release |
      | traderGuy | ETH/DEC19 |    98600008 | 108460008 | 118320009 | 138040011 |
# add more orders
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |      2 | 30000000 |                0 | LIMIT | GTC |
      | sellSideProvider | ETH/DEC19 | sell |      2 | 20000000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      2 | 19500000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      2 |  8000000 |                0 | LIMIT | GTC |
# checking margins
    Then the margins levels for the traders are:
      | trader    | id        | maintenance |    search |   initial |   release |
      | traderGuy | ETH/DEC19 |    98600008 | 108460008 | 118320009 | 138040011 |
