Feature: Position resolution case 4

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  BTC        | BTC   |  simple     | 0         | 0         | 0              | 0.016           | 2.0   | 5              | 4              | 3.2           | 42               | 0                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 10000         |

# insurance pool generation - setup orderbook
    Then traders place following orders with references:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 190   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 180   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

# insurance pool generation - trade
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     |
      | designatedLooser | ETH/DEC19 | sell | 100    | 180   | 2                | TYPE_LIMIT | TIF_GTC |

    Then the margins levels for the traders are:
      | trader           | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 2000        | 6400   | 8000    | 10000   |

# insurance pool generation - modify order book
    Then traders cancel the following orders:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |

# add back some volume on the sell side
    Then traders place following orders with references:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 350   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |

# insurance pool generation - set new mark price (and trigger closeout)
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 300   | 1                | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "300"

#check positions
    Then position API produce the following:
      | trader           | volume | unrealisedPNL | realisedPNL |
      | designatedLooser | 0      | 0             | -10000      |
      | buySideProvider  | 101    | 11500         | -1500       |

# checking margins
    Then traders have the following account balances:
      | trader           | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 0      | 0       |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance is "0" for the market "ETH/DEC19"


# now we check what's left in the orderbook
# we expect 50 orders to be left there on the sell side
# we buy a first time 50 to consume the book
# then try to buy 1 again -> result in no trades -> sell side empty.
# Try to sell one for low price -> no trades -> buy side empty -> order book empty.
    Then traders place following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 350   | 1                | TYPE_LIMIT | TIF_FOK |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 350   | 0                | TYPE_LIMIT | TIF_FOK |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 1     | 0                | TYPE_LIMIT | TIF_FOK |
