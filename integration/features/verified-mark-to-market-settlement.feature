Feature: MTM settlement tests
# Reference spreadsheet: https://drive.google.com/open?id=1ZCj7WWvP236wiJDgiGD_f9Xsun9o8PsW
  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |

  Scenario: case 1 - LONG - MORE LONG - one trade
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade for 10@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

# place trade for 1@111 to set new mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "111" for the market "ETH/DEC19"

# MTM win transfers: 200+30=230 as per spreadsheet
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 30     | BTC   |


  Scenario: case 2 - LONG - MORE LONG - multiple trades
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for 10@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# place trade 2 for 2@113
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 2      | 113   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 2      | 113   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "113" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 90     | BTC   |

# place trade for 1@111 to set new mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "111" for the market "ETH/DEC19"


# MTM win transfers: 200+90-64=226 as per spreadsheet
    Then the following transfers should happen:
      | from    | to     | from account        | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 64     | BTC   |

  Scenario: case 3 - LONG - LESS LONG - one trade
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -5@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 5      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 5      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

# place trade for 1@111 to set new mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "111" for the market "ETH/DEC19"

# MTM win transfers: 200+15=215 as per spreadsheet
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 15     | BTC   |


  Scenario: case 4 - LONG - LESS LONG - multiple trades
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -10@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# place trade 2 for -2@113
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 2      | 113   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 2      | 113   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "113" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 30     | BTC   |

# place trade for 1@111 to set new mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "111" for the market "ETH/DEC19"

# MTM win transfers: 200+30-16=214 as per spreadsheet
    Then the following transfers should happen:
      | from    | to     | from account        | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 16     | BTC   |

  Scenario: case 5 - LONG - ZERO - one trade
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -20@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 20     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 20     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

  Scenario: case 6 - LONG - ZERO - multiple trades
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -10@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# place trade 2 for -10@113
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 10     | 114   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 10     | 114   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "114" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 40     | BTC   |

  Scenario: case 7 - LONG - SHORT - one trade
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -30@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 30     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 30     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

  Scenario: case 7 - LONG - SHORT - multiple trades
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for 5@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 5      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 5      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# place trade 2 for -30@114
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 30     | 114   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 30     | 114   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "114" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 100    | BTC   |

  # place trade for 1@111 to set new mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "111" for the market "ETH/DEC19"

# MTM win transfers: 200+100+15=315 as per spreadsheet
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 15     | BTC   |

  Scenario: case 8 - LONG - SAME AMOUNT - multiple trades
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000000 |
      | party5 | BTC   | 10000000 |
      | aux     | BTC   | 100000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 10     | 115   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party4-1 |
      | party5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party5-1 |
      | party4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party4-2 |
      | party5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party5-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | party4-1 |
      | party5 | party5-1 |

# setup previous volume at 20
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for 10@110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "110" for the market "ETH/DEC19"

# place trade 2 for -10@114
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 10     | 114   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 10     | 114   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "114" for the market "ETH/DEC19"

# MTM win transfers
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 120    | BTC   |

# place trade for 1@111 to set new mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "111" for the market "ETH/DEC19"

# MTM win transfers: 200+120-60=260 as per spreadsheet
    Then the following transfers should happen:
      | from    | to     | from account        | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 60     | BTC   |
