Feature: Position resolution case 3

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 5              | 4              | 3.2           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: https://docs.google.com/spreadsheets/d/1D433fpt7FUCk04dZ9FHDVy-4hA6Bw_a2
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 12000         |
      | aux              | BTC   | 100000        |

# insurance pool generation - setup orderbook
    Then traders place following orders with references:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

# insurance pool generation - trade
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     |
      | designatedLooser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC |

    Then the margins levels for the traders are:
      | trader           | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 2900        | 9280   | 11600   | 14500   |

# insurance pool generation - modify order book
    Then traders cancel the following orders:
      | trader          | reference      |
      | buySideProvider | buy-provider-1 |
    Then traders place following orders with references:
      | trader          | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 300    | 40    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

# check the trader accounts
    Then traders have the following account balances:
      | trader           | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 11600  | 400     |

# insurance pool generation - set new mark price (and trigger closeout)
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC |

# check positions
    Then traders have the following profit and loss:
      | trader           | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -12000       |

# checking margins
    Then traders have the following account balances:
      | trader           | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 0      | 0       |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance is "3300" for the market "ETH/DEC19"


     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume  | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1       |     1 | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1       |  1001 | 0                | TYPE_LIMIT  | TIF_GTC | 

# now we check what's left in the orderbook
# we expect 10 orders at price of 40 to be left there on the buy side
# we sell a first time 10 to consume the book
# then try to sell 1 again with low price -> result in no trades -> buy side empty
# We expect no orders on the sell side: try to buy 1 for high price -> no trades -> sell side empty
   Then traders place following orders:
      | trader           | id          | type  | volume | price | resulting trades  | type  | tif |
      | sellSideProvider | ETH/DEC19   | sell  |     10 |   40  |                1  | TYPE_LIMIT | TIF_FOK |
      | sellSideProvider | ETH/DEC19   | sell  |     1  |    2  |                0  | TYPE_LIMIT | TIF_FOK |
      | buySideProvider  | ETH/DEC19   | buy   |     1  | 1000  |                0  | TYPE_LIMIT | TIF_FOK |
