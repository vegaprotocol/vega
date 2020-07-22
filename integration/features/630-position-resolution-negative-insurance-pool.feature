Feature: Regression test for issue 630

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |        0.2 |      0.1 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |

  Scenario: Trader is being closed out.
# setup accounts
    Given the following traders:
      | name             |  amount |
      | sellSideProvider | 1000000 |
      | buySideProvider  | 1000000 |
      | traderGuy        |  240000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | traderGuy        | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
# setup orderbook
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |    200 | 10000 |                0 | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  |    200 |     1 |                0 | TYPE_LIMIT | TIF_GTC |
    And All balances cumulated are worth "2240000"
    Then the margins levels for the traders are:
      | trader           | id        | maintenance | search | initial | release |
      | sellSideProvider | ETH/DEC19 |        2000 |   2200 |    2400 |    2800 |
    Then traders place following orders:
      | trader    | id        | type      | volume | price | resulting trades | type  | tif |
      | traderGuy | ETH/DEC19 | buy       |    100 | 10000 |                1 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
     | trader           | asset | id        | margin | general |
     | traderGuy        | BTC   | ETH/DEC19 |      0 |  0 |
     | sellSideProvider | BTC   | ETH/DEC19 | 240000 |  760000 |
    And the insurance pool balance is "240000" for the market "ETH/DEC19"
    And All balances cumulated are worth "2240000"
