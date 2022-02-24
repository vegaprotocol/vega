Feature: Position resolution case 3

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: set "designatedLooser" to be closed out since there is enough vol on the order book to take over 
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 12000         |
      | aux              | BTC   | 1000000000000 |
      | aux2             | BTC   | 1000000000000 |

# insurance pool generation - setup orderbook
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 291    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux              | ETH/DEC19 | sell | 1      | 155   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# insurance pool generation - trade
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the order book should have the following volumes for market "ETH/DEC19":   
      | side | price  | volume |
      | buy  | 140    | 2      |
     
  #designatedLooser has position of vol 290; price 150; RiskFactor is 0; 
  #what's on the order book to cover the position is shown above, which makes the exit price 70 =(140*2)/2, slippage per unit is 150-140=10
  #margin level is PositionVol*SlippagePerUnit = 290*10 = 2900
    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 2900        | 9280   | 11600   | 14500   |
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 11600  | 400     |

# insurance pool generation - modify order book
    Then the parties cancel the following orders:
      | party          | reference      |
      | buySideProvider | buy-provider-1 |
    When the parties place the following orders:
      | party          | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 300    | 40    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

    Then the order book should have the following volumes for market "ETH/DEC19":   
      | side | price  | volume |
      | buy  | 140    | 1      |
      | buy  | 40     | 300    |

# check the party accounts, change of order on the order book will not trigger margin recalcution
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 11600  | 400     |

# insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      
#"designatedLooser" is closed out since there is enough order on the book to cover "designatedLooser"'s position which is 290
# check positions
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -12000       |
      | buySideProvider  | 290    | 29000          | -19900       |

# checking margins
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 0      | 0       |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "0" for the market "ETH/DEC19"


    
