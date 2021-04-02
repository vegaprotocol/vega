Feature: Position resolution case 3

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

  Scenario: https://docs.google.com/spreadsheets/d/1D433fpt7FUCk04dZ9FHDVy-4hA6Bw_a2
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 12000         |
      | aux              | BTC   | 1000000000000 |
      | aux2             | BTC   | 1000000000000 |

# insurance pool generation - setup orderbook
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 291    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux              | ETH/DEC19 | sell | 1      | 155   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "150"
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

# insurance pool generation - trade
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the margins levels for the traders are:
      | trader           | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 2900        | 9280   | 11600   | 14500   |

# insurance pool generation - modify order book
    Then traders cancel the following orders:
      | trader          | reference      |
      | buySideProvider | buy-provider-1 |
    When traders place the following orders:
      | trader          | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 300    | 40    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

# check the trader accounts
    Then traders have the following account balances:
      | trader           | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 11600  | 400     |

# insurance pool generation - set new mark price (and trigger closeout)
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# check positions
    Then traders have the following profit and loss:
      | trader           | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -12000       |

# checking margins
    Then traders have the following account balances:
      | trader           | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 0      | 0       |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance is "9100" for the market "ETH/DEC19"


     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume  | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1       |     1 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1       |  1001 | 0                | TYPE_LIMIT  | TIF_GTC |

# now we check what's left in the orderbook
# we expect 10 orders at price of 40 to be left there on the buy side
# we sell a first time 10 to consume the book
# then try to sell 1 again with low price -> result in no trades -> buy side empty
# We expect no orders on the sell side: try to buy 1 for high price -> no trades -> sell side empty
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 10     | 40    | 1                | TYPE_LIMIT | TIF_FOK | ref-1     |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_FOK | ref-3     |
