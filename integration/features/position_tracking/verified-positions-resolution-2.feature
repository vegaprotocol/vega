Feature: Position resolution case 2

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: https://docs.google.com/spreadsheets/d/1D433fpt7FUCk04dZ9FHDVy-4hA6Bw_a2/edit#gid=1011478143
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 12000         |
      | aux              | BTC   | 1000000000000 |
      | aux2             | BTC   | 1000000000000 |

# place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 151   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# insurance pool generation - setup orderbook
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

# insurance pool generation - trade
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the order book should have the following volumes for market "ETH/DEC19":   
      | side | price  | volume |
      | buy  | 1      | 1      |
      | buy  | 140    | 1      |

  #designatedLooser has position of vol 290; price 150; RiskFactor is 0; 
  #what's on the order book to cover the position is shown above, which makes the exit price 70 =(1*1+141*1)/2, slippage per unit is 150-70=80
  #margin level is PositionVol*SlippagePerUnit = 290*80 = 23200
    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search  | initial  | release |
      | designatedLooser | ETH/DEC19 | 23200       | 74240   | 92800    | 116000  |

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |  
      | designatedLooser | BTC   | ETH/DEC19 | 12000  | 0       |

# insurance pool generation - modify order book
    Then the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 1      | 40    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

# change of order on order book will  not trigger recalculation of margin level of "designatedLooser", so margin account stays unchanged
# check the party accounts
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |  
      | designatedLooser | BTC   | ETH/DEC19 | 12000  | 0       |

# insurance pool generation - set new mark price (and trigger closeout)
# the exit price 70 =(1*1+141*1)/2, slippage per unit is 120-70=50
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the order book should have the following volumes for market "ETH/DEC19":   
      | side | price  | volume |
      | buy  | 1      | 1      |
      | buy  | 40     | 1      |

# change of mark price triggered recalculation of margin level of "designatedLooser" 
# since the change of order before the change of mark price, the slippage is recalculated as well: 120-(1*1+1*40)/2 = 100, hence margin level is 100*290 = 29000
    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search  | initial  | release |
      | designatedLooser | ETH/DEC19 | 29000       | 92800   | 116000   | 145000  |

 #margin account got MTM (since the change of mark price from 150 to 120), hence margin account = 12000 - 290*30 = 3300
 # ??????should "designatedLooser" be closeout at this point??????? new mark price is on line 79, and margin account is much lower than maintence level

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |  
      | designatedLooser | BTC   | ETH/DEC19 | 3300   | 0       |

# check positions
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 290    | -8700          | 0            |

# checking margins
    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 3300   | 0       |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "0" for the market "ETH/DEC19"

# now we check what's left in the orderbook
# we expect 1 order at price of 40 to be left there on the buy side
# we sell a first time 1 to consume the book
# then try to sell 1 again with low price -> result in no trades -> buy side empty
# We expect no orders on the sell side: try to buy 1 for high price -> no trades -> sell side empty
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 40    | 1                | TYPE_LIMIT | TIF_FOK | ref-1     |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_FOK | ref-3     |
