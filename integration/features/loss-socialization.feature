Feature: Test loss socialization behaviour

  Background:
    Given the insurance pool initial balance for the markets is "0":
 #   And the markets starts on "2019-11-30T00:00:00Z" and expires on "2019-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |

  Scenario: Test loss socialization
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
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   120 |                0 | LIMIT | GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | LIMIT | GTC | buy-provider-1  |
# trader 1 place an order + we check margins
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |    100 |   100 |                0 | LIMIT | GTC |
    # Then the margins levels for the traders are:
    #   | trader  | id        | maintenance | search | initial | release |
    #   | trader1 | ETH/DEC19 |        1600 |   1760 |    1920 |    2240 |
    # Then I expect the trader to have a margin:
    #  | trader  | asset | id        | margin | general |
    #  | trader1 | BTC   | ETH/DEC19 |   1920 |    3080 |
# then trader2 place an order, and we calculate the margins again
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader2 | ETH/DEC19 | buy  |    100 |   100 |                1 | LIMIT | GTC |
# trade happened, no we check the traders margins, and update the state of the book
    # Then the margins levels for the traders are:
    #   | trader  | id        | maintenance | search | initial | release |
    #   | trader2 | ETH/DEC19 |        2000 |   3300 |    3600 |    4200  |
    Then I expect the trader to have a margin:
     | trader  | asset | id        | margin | general |
     | trader1 | BTC   | ETH/DEC19 |   2400 |    2600 |
#    | trader2 | BTC   | ETH/DEC19 |   2400 |    7600 |
# then we change the volume in the book
    Then traders cancels the following orders reference:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   200 |                0 | LIMIT | GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | LIMIT | GTC | buy-provider-2  |
    And the insurance pool balance is "0" for the market "ETH/DEC19"
    And All balances cumulated are worth "200105000"

    Then I expect the trader to have a margin:
     | trader  | asset | id        | margin | general |
     | trader1 | BTC   | ETH/DEC19 |   2400 |    2600 |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader2 | ETH/DEC19 | buy  |    100 |   180 |                0 | LIMIT | GTC |
      | trader3 | ETH/DEC19 | sell |    100 |   180 |                1 | LIMIT | GTC |

    And dump transfers

    # Then the following transfers happened:
    #   | from    | to     | fromType | toType     | id        | amount | asset |
    #   | trader1 | market | MARGIN   | SETTLEMENT | ETH/DEC19 |    240 | BTC   |

    Then I expect the trader to have a margin:
     | trader  | asset | id        | margin | general |
     | trader1 | BTC   | ETH/DEC19 |      0 |       0 |





    # And the mark price is "180"

# then check the position API, change time first in order to flush the stores and get data propagated
#    Then the time is updated to "2020-01-01T01:01:01Z"
    Then position API produce the following:
      | trader  | volume | realisedPNL | unrealisedPNL |
      | trader1 |      0 |       -5000 |             0 |
      | trader2 |    200 |           0 |          5000 |
      | trader3 |   -100 |           0 |             0 |
