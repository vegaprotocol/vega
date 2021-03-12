Feature: MTM settlement tests
# Reference spreadsheet: https://drive.google.com/open?id=1ZCj7WWvP236wiJDgiGD_f9Xsun9o8PsW
  Background:
    Given the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |


  Scenario: case 1 - LONG - MORE LONG - one trade
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader3 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade for 10@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+30=230 as per spreadsheet
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 30     | BTC   |


  Scenario: case 2 - LONG - MORE LONG - multiple trades
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader3 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for 10@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for 2@113
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 2      | 113   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 2      | 113   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "113"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 90     | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "111"


# MTM win transfers: 200+90-64=226 as per spreadsheet
    Then the following transfers happened:
      | from    | to     | fromType             | toType                  | id        | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |     64 | BTC   |

  Scenario: case 3 - LONG - LESS LONG - one trade
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader3 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -5@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 5      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 5      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+15=215 as per spreadsheet
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 15     | BTC   |


  Scenario: case 4 - LONG - LESS LONG - multiple trades
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader3 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -10@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -2@113
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 2      | 113   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 2      | 113   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "113"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 30     | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+30-16=214 as per spreadsheet
    Then the following transfers happened:
      | from   | to      | fromType             | toType                  | id        | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |     16 | BTC   |

  Scenario: case 5 - LONG - ZERO - one trade
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -20@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 20     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 20     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

  Scenario: case 6 - LONG - ZERO - multiple trades
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -10@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -10@113
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 10     | 114   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 10     | 114   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 40     | BTC   |

  Scenario: case 7 - LONG - SHORT - one trade
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for -30@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 30     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 30     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |

  Scenario: case 7 - LONG - SHORT - multiple trades
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader3 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for 5@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 5      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -30@114
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 30     | 114   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 30     | 114   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 100    | BTC   |

  # place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+100+15=315 as per spreadsheet
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 15     | BTC   |

  Scenario: case 8 - LONG - SAME AMOUNT - multiple trades
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount   |
      | trader1 | BTC   | 10000    |
      | trader2 | BTC   | 10000    |
      | trader3 | BTC   | 10000    |
      | trader4 | BTC   | 10000000 |
      | trader5 | BTC   | 10000000 |
      | aux     | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 10      |   99  | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 10      |  115  | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader5 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader5-1 |
      | trader4 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
      | trader5 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader5-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader4 | trader4-1 |
      | trader5 | trader5-1 |

# setup previous volume at 20
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 20     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# place trade 1 for 10@110
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 10     | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "110"

# place trade 2 for -10@114
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 10     | 114   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 10     | 114   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "114"

# MTM win transfers
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 200    | BTC   |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 120    | BTC   |

# place trade for 1@111 to set new mark price
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | sell | 1      | 111   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "111"

# MTM win transfers: 200+120-60=260 as per spreadsheet
    Then the following transfers happened:
      | from   | to      | fromType             | toType                  | id        | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |     60 | BTC   |
