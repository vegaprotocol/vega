Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |        94 | simple     |        0.2 |      0.1 |  0 | 0.016 |   2.0 |              5 |              4 |           3.2 |              42 |

  Scenario: Traded out position but monies left in margin account
# setup accounts
    Given the following traders:
      | name             |     amount |
      | traderGuy        | 1000000000 |
      | sellSideProvider | 1000000000 |
      | buySideProvider  | 1000000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | traderGuy        | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
# setup previous mark price
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |      1 | 10300000 |                0 | LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 | 10300000 |                1 | LIMIT | TIF_GTC |
# setup orderbook
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |    100 | 25000000 |                0 | LIMIT | TIF_GTC |
      | sellSideProvider | ETH/DEC19 | sell |     11 | 14000000 |                0 | LIMIT | TIF_GTC |
      | sellSideProvider | ETH/DEC19 | sell |      2 | 11200000 |                0 | LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 | 10000000 |                0 | LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      3 |  9600000 |                0 | LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  |     15 |  9000000 |                0 | LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  |     50 |  8700000 |                0 | LIMIT | TIF_GTC |
# buy 13@150
    Then traders place following orders:
      | trader    | id        | type | volume |    price | resulting trades | type  | tif |
      | traderGuy | ETH/DEC19 | buy  |     13 | 15000000 |                2 | LIMIT | TIF_GTC |
# checking margins
    Then I expect the trader to have a margin:
      | trader    | asset | id        |    margin |   general |
      | traderGuy | BTC   | ETH/DEC19 | 394400032 | 611199968 |
# checking margins levels
    Then the margins levels for the traders are:
      | trader    | id        | maintenance |    search |   initial |   release |
      | traderGuy | ETH/DEC19 |    98600008 | 315520025 | 394400032 | 493000040 |
