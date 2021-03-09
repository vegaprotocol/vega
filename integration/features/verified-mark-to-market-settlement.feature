Feature: MTM settlement tests
# Reference spreadsheet: https://drive.google.com/open?id=1ZCj7WWvP236wiJDgiGD_f9Xsun9o8PsW
  Background:
    Given the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  BTC        | BTC   | 100        | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 42               | 0                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: case 1 - LONG - MORE LONG - one trade
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
      | trader3          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade for 10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     10 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     10 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | buy  |     1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     1 |   111 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+30=230 as per spreadsheet
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    30 | BTC   |


  Scenario: case 2 - LONG - MORE LONG - muliple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
      | trader3          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for 10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     10 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     10 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for 2@113
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |      2 |   113 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |      2 |   113 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "113"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |     90 | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | buy  |     1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     1 |   111 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+90-64=226 as per spreadsheet
    Then the following transfers happened:
      | from    | to     | fromType             | toType                  | id        | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |     64 | BTC   |

  Scenario: case 3 - LONG - LESS LONG - one trade
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
      | trader3          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for -5@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |      5 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |      5 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | buy  |     1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     1 |   111 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+15=215 as per spreadsheet
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    15 | BTC   |


  Scenario: case 4 - LONG - LESS LONG - multiple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
      | trader3          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for -10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -2@113
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |      2 |   113 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |      2 |   113 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "113"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |     30 | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | buy  |     1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     1 |   111 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+30-16=214 as per spreadsheet
    Then the following transfers happened:
      | from   | to      | fromType             | toType                  | id        | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |     16 | BTC   |

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
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for -20@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     20 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     20 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |

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
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for -10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -10@113
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   114 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   114 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |     40 | BTC   |

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
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for -30@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     30 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     30 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |

  Scenario: case 7 - LONG - SHORT - multiple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
      | trader3          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for 5@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |      5 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |      5 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -30@114
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     30 |   114 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     30 |   114 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    100 | BTC   |

  # place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | buy  |     1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     1 |   111 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+100+15=315 as per spreadsheet
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    15  | BTC   |

  Scenario: case 8 - LONG - SAME AMOUNT - multiple trades
# setup accounts
    Given the following traders:
      | name             |    amount |
      | trader1          |     10000 |
      | trader2          |     10000 |
      | trader3          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |

    And the mark price for the market "ETH/DEC19" is "100"

# setup previous volume at 20
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     20 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     20 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# place trade 1 for 10@110
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  |     10 |   110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     10 |   110 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -10@114
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |   114 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     10 |   114 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    200 | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 |    120 | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | buy  |     1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell |     1 |   111 |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+120-60=260 as per spreadsheet
    Then the following transfers happened:
      | from   | to      | fromType             | toType                  | id        | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |     60 | BTC   |
