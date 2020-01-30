Feature: MTM settlement tests

  Background:
    Given the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |

  Scenario: case 1 - LONG - MORE LONG - one trade
# setup accounts
    Given the following traders:
      | name             |    amount |
      | sellSideProvider | 100000000 |
      | buySideProvider  | 100000000 |
      | trader1          |     10000 |
      | trader2          |     10000 |
      | trader3          |     10000 |
      | trader4          |     10000 |
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

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade for 10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     10 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     10 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |
