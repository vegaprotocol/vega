Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short |               tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       103 | forward    |      0.001 | 0.00011407711613050422 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |

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
      | sellSideProvider | ETH/DEC19 | sell |      1 | 10300000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 | 10300000 |                1 | LIMIT | GTC |
# setup orderbook
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
    Then I expect the trader to have a margin:
      | trader    | asset | id        |   margin |   general |
      | traderGuy | BTC   | ETH/DEC19 | 97880181 | 907719819 |
