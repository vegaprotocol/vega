Feature: MTM settlement tests

  Background:
    Given the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 |

  Scenario: case 1 - LONG - MORE LONG - one trade
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

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

  Scenario: case 2 - LONG - MORE LONG - muliple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for 10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     10 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     10 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for 2@113
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |      2 |   113 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |      2 |   113 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "113"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |     90 | BTC   |

  Scenario: case 3 - LONG - LESS LONG - one trade
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for -5@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |      5 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |      5 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |

  Scenario: case 4 - LONG - LESS LONG - multiple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for -10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -2@113
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |      2 |   113 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |      2 |   113 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "113"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |     30 | BTC   |

  Scenario: case 5 - LONG - ZERO - one trade
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for -20@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     20 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |     20 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |

  Scenario: case 6 - LONG - ZERO - multiple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for -10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -10@113
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   114 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   114 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |     40 | BTC   |

  Scenario: case 7 - LONG - SHORT - one trade
# setup accounts
    Given the following traders:
      | name    | amount |
      | trader1 |  10000 |
      | trader2 |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | BTC   |
      | trader2 | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for -30@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     30 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |     30 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |

  Scenario: case 7 - LONG - SHORT - multiple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for 5@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |      5 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |      5 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -30@114
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     30 |   114 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |     30 |   114 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    100 | BTC   |

  Scenario: case 8 - LONG - SAME AMOUNT - multiple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | LIMIT | GTC |

# place trade 1 for 10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     10 |   110 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | sell |     10 |   110 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -10@114
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   114 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   114 |                1 | LIMIT | GTC |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    120 | BTC   |
