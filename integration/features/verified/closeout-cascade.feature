Feature: Closeout-cascades
# This is a test case to demonstrate that closeout cascade does NOT happen. In this test case, trader3 gets closed
# out first after buying volume 50 at price 100, and then trader3's position is sold (via the network counterparty) to trader2 whose order is first in the order book 
# (volume 50 price 50)
# At this moment, trader2 is under margin but the fuse breaks (if the mark price is actually updated from 100 to 50)
# however, the design of the system is like a fuse and circuit breaker, the mark price will NOT update, so trader2 will not be closed out
# till a new trade happens, and new mark price is set.
  Background:

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5           | 2              | 3              | 
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator   | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: Distressed position gets taken over by another party whose margin level is insufficient to support it (however mark price doesn't get updated on closeout trade and hence no further closeouts are carried out) 
   # setup accounts, we are trying to closeout trader3 first and then trader2
    
    Given the insurance pool balance should be "0" for the market "ETH/DEC19"
    
    Given the parties deposit on asset's general account the following amount:
      | party        | asset | amount        |
      | auxiliary1   | BTC   | 1000000000000 |
      | auxiliary2   | BTC   | 1000000000000 |
      | trader2      | BTC   | 150           |
      | trader3      | BTC   | 100           |

    Then the cumulated balance for all accounts should be worth "2000000000250"

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
  # trading happens at the end of the open auction period 
    Then the parties place the following orders:
      | party     | market id | side | volume | price  | resulting trades| type       | tif     | reference |
      | auxiliary2| ETH/DEC19 | buy  | 5      | 5      | 0               | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1| ETH/DEC19 | sell | 10     | 1000   | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1000|
      | auxiliary2| ETH/DEC19 | buy  | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the auction ends with a traded volume of "10" at a price of "10"
    And the mark price should be "10" for the market "ETH/DEC19"

    And the cumulated balance for all accounts should be worth "2000000000250"

    # setup trader2 position to be ready to takeover trader3's position once trader3 is closed out
    When the parties place the following orders: 
      | party     | market id | side | volume| price | resulting trades | type       | tif     | reference      |
      | trader2   | ETH/DEC19 | buy  | 50    | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-position-3 |
     
    And the parties should have the following margin levels:
      | party     | market id  | maintenance | search | initial | release |
      | trader2   | ETH/DEC19  | 50          | 75     | 100     | 150     |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin   | general |
      | trader2 | BTC   | ETH/DEC19 | 100      | 50      |
    
    And the cumulated balance for all accounts should be worth "2000000000250"

    # setup trader3 position and close it out
    When the parties place the following orders: 
      | party      | market id | side | volume| price | resulting trades | type       | tif     | reference      |
      | trader3    | ETH/DEC19 | buy  | 50    | 100   | 0                | TYPE_LIMIT | TIF_GTC | buy-position-3 |
      | auxiliary1 | ETH/DEC19 | sell | 50    | 100   | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1|
      | auxiliary2 | ETH/DEC19 | buy  | 50    | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1|

    And the mark price should be "100" for the market "ETH/DEC19"
       Then debug transfers
    # trader3 got close-out, trader3's order has been sold to network, and then trader2 bought the order from the network 
    # as it had the highest buy price 
    And the following trades should be executed:
      | buyer   | price | size | seller  | 
      | network |  50   | 50   | trader3 | 
      | trader2 |  50   | 50   | network | 

    And the mark price should be "100" for the market "ETH/DEC19"

    And the cumulated balance for all accounts should be worth "2000000000250"

    # check that trader3 is closed-out but trader2 is not 
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 50          | 75     | 100     | 150     |
      | trader3 | ETH/DEC19 | 0           | 0      | 0       | 0       |
     Then the parties should have the following profit and loss: 
      | party   | volume | unrealised pnl | realised pnl |
      | trader2 | 50     | 2500           | -2400        |
      | trader3 | 0      | 0              | -100         |

    # check trader2 margin level, trader2 is not closed-out yet since new mark price is not updated
    # eventhough  trader2 does not have enough margin 
    Then the parties should have the following account balances:
      | party      | asset | market id | margin    | general      |
      | trader2    | BTC   | ETH/DEC19 | 200       | 50           |
      | trader3    | BTC   | ETH/DEC19 | 0         | 0            |
      | auxiliary1 | BTC   | ETH/DEC19 | 109400    | 999999889700 |
      | auxiliary2 | BTC   | ETH/DEC19 | 3200      | 999999997700 |
     
    # setup new mark price, which is the same as when trader2 traded with network
    Then the parties place the following orders:
      | party     | market id | side | volume | price  | resulting trades| type       | tif     | reference |
      | auxiliary2| ETH/DEC19 | buy  | 10     | 50     | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10     | 50     | 1               | TYPE_LIMIT | TIF_GTC | aux-s-1   |

    And the mark price should be "50" for the market "ETH/DEC19"

    #trader2 got closed-out 
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader2 | 0      | 0              | -150         |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 0           | 0      | 0       | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    Then the parties should have the following profit and loss:
      | party           | volume | unrealised pnl | realised pnl |
      | trader2         | 0      |     0          | -150         |
      | trader3         | 0      |     0          | -100         | 
      | auxiliary1      | -70    |  2100          | -2250        |
      | auxiliary2      |  70    |  2400          | -2000        |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | sell | 1000     | 10     |
      | sell | 100      | 0      |
      | sell | 50       | 0      |
      | buy  | 50       | 0      |
      | buy  | 10       | 0      |
      | buy  | 5        | 5      |
      
    
