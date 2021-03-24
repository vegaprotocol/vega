Feature: Position resolution case 4

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
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
      | auxiliary        | BTC   | 1000000000000 |
      | auxiliary2       | BTC   | 1000000000000 |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
  Then traders place the following orders:
    | trader     | market id | side | volume | price   | resulting trades | type        | tif     | reference |
    | auxiliary2 | ETH/DEC19 | buy  | 1      | 1       | 0                | TYPE_LIMIT  | TIF_GTC | aux-b-1   |
    | auxiliary  | ETH/DEC19 | sell | 1      | 1000    | 0                | TYPE_LIMIT  | TIF_GTC | aux-s-1   |
    | auxiliary  | ETH/DEC19 | sell | 10     | 180     | 0                | TYPE_LIMIT  | TIF_GTC | aux-s-2   |
    | auxiliary2 | ETH/DEC19 | buy  | 10     | 180     | 0                | TYPE_LIMIT  | TIF_GTC | aux-b-2   |
    Then the opening auction period for market "ETH/DEC19" ends
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC19" is "180"

# insurance pool generation - setup orderbook
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 190   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 180   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

# insurance pool generation - trade
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | sell | 100    | 180   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the margins levels for the traders are:
      | trader           | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 2000        | 6400   | 8000    | 10000   |

# insurance pool generation - modify order book
    Then traders cancel the following orders:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |

# add back some volume on the sell side
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 350   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |

# insurance pool generation - set new mark price (and trigger closeout)
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 300   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC19" is "300"

#check positions
    Then traders have the following profit and loss:
      | trader           | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -10000       |
      | buySideProvider  | 101    | 11500          | -1363        |

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
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 350   | 1                | TYPE_LIMIT | TIF_FOK | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 350   | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_FOK | ref-3     |
