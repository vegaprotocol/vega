Feature: Position resolution case 4

  Background: "designatedLooser" is set to be closed out with enough order on the book to take over its position

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: 
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 10000         |
      | auxiliary        | BTC   | 1000000000000 |
      | auxiliary2       | BTC   | 1000000000000 |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary2 | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | auxiliary  | ETH/DEC19 | sell | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
      | auxiliary2 | ETH/DEC19 | buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "180" for the market "ETH/DEC19"

# insurance pool generation - setup orderbook
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 190   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 180   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

# insurance pool generation - trade
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | sell | 100    | 180   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |

  Then the order book should have the following volumes for market "ETH/DEC19":   
      | side | price  | volume |
      | buy  | 1      | 1      |
      | sell | 200    | 150    |
      | sell | 1000   | 1      |

# exit price for Looser is 200, traded price for looser is (180*50+180*50)/100=180, so slippage per unit is 20
# margin level is vol*slippage = 100 * 20 = 20000
    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 2000        | 6400   | 8000    | 10000   |

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 8000   | 2500    |

# insurance pool generation - modify order book
    Then the parties cancel the following orders:
      | party            | reference       |
      | sellSideProvider | sell-provider-1 |

# add back some volume on the sell side
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 350   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |

# insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 300   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "300" for the market "ETH/DEC19"

    #hance margin MTM for designatedLooser is 100*(300-100)=20000 which is larger than its collateral, so designatedLooser is closed out

#check positions
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -10000       |
      | buySideProvider  | 101    | 11500          | -1363        |
      | sellSideProvider | -101   | 5000           | -5000        |

# checking margins
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 0      | 0       |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "0" for the market "ETH/DEC19"



