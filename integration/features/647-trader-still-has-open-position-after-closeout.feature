Feature: Regression test for 647

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |    150000 | simple     |       0.03 |     0.02 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |

  Scenario: Trader is being closed out.
# setup accounts
    Given the following traders:
      | name    |   amount |
      | trader1 | 50000000 |
      | trader2 |   139200 |
      | trader3 | 50000000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | BTC   |
      | trader2 | BTC   |
      | trader3 | BTC   |
# setup orderbook
    Then traders place following orders:
      | trader  | id        | type | volume |  price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     20 | 170000 |                0 | LIMIT | GTC |
      | trader1 | ETH/DEC19 | sell |     10 | 160000 |                0 | LIMIT | GTC |
      | trader1 | ETH/DEC19 | buy  |     10 | 140000 |                0 | LIMIT | GTC |
      | trader1 | ETH/DEC19 | buy  |     20 | 135000  |                0 | LIMIT | GTC |
    And All balances cumulated are worth "100139200"
    Then traders place following orders:
      | trader  | id        | type | volume |  price | resulting trades | type  | tif |
      | trader2 | ETH/DEC19 | buy  |      5 | 160000 |                1 | LIMIT | GTC |
    And All balances cumulated are worth "100139200"
    Then traders place following orders:
      | trader  | id        | type | volume |  price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | sell |     15 | 130000 |                2 | LIMIT | GTC |
