Feature: Test loss socialization case 1

  Background:
    Given the insurance pool initial balance for the markets is "0":
 #   And the markets starts on "2019-11-30T00:00:00Z" and expires on "2019-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |

  Scenario: case 1 from https://docs.google.com/spreadsheets/d/1CIPH0aQmIKj6YeFW9ApP_l-jwB4OcsNQ/edit#gid=1555964910
# setup accounts
    Given the following traders:
      | name             |    amount |
      | sellSideProvider | 100000000 |
      | buySideProvider  | 100000000 |
      | trader1          |      5000 |
      | trader2          |     50000 |
      | trader3          |     50000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
# setup orderbook
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
# trader 1 place an order + we check margins
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |    100 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
# then trader2 place an order, and we calculate the margins again
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader2 | ETH/DEC19 | buy  |    100 |   100 |                1 | TYPE_LIMIT | TIF_GTC |
# then we change the volume in the book
    Then traders cancels the following orders reference:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   200 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-2  |
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader2 | ETH/DEC19 | buy  |    100 |   180 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC19 | sell |    100 |   180 |                1 | TYPE_LIMIT | TIF_GTC |
    Then position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |      0 |             0 |       -5000 |
      | trader2 |    200 |          8000 |       -3000 |
      | trader3 |   -100 |             0 |           0 |
    And the insurance pool balance is "0" for the market "ETH/DEC19"
    And All balances cumulated are worth "200105000"
