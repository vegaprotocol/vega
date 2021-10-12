Feature: closeout-cascases & https://github.com/vegaprotocol/vega/pull/4138/files)

  Background:

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5             | 2              | 3              | 
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator   | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: Distressed position gets taken over by another party whose margin level is insufficient to support it 
   # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party        | asset | amount        |
      | trader1      | BTC   | 800           |
      | trader2      | BTC   | 150           |
      | trader3      | BTC   | 100           |
      | auxiliary1   | BTC   | 1000000000000 |
      | auxiliary2   | BTC   | 1000000000000 |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price  | resulting trades| type       | tif     | reference |
      | auxiliary2| ETH/DEC19 | buy  | 5      | 5      | 0               | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | auxiliary1| ETH/DEC19 | sell | 10     | 1000   | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | auxiliary2| ETH/DEC19 | buy  | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the auction ends with a traded volume of "10" at a price of "10"
    And the mark price should be "10" for the market "ETH/DEC19"

# setup trader3 position and close it out
    When the parties place the following orders: 
      | party     | market id | side | volume| price | resulting trades | type       | tif     | reference       |
      | trader2   | ETH/DEC19 | buy  | 50   | 50    | 0                  | TYPE_LIMIT | TIF_GTC | buy-position-3 |
     
    And the parties should have the following margin levels:
      | party | market id   | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 50       | 75  | 100 | 150 |
     
     When the parties place the following orders: 
      | party     | market id | side | volume| price | resulting trades | type       | tif     | reference       |
      | trader3   | ETH/DEC19 | buy  | 50   | 100    | 0                | TYPE_LIMIT | TIF_GTC | buy-position-3 |
      | auxiliary1| ETH/DEC19 | sell | 50   | 100    | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1|
      | auxiliary2| ETH/DEC19 | buy  | 50   | 10    | 0               | TYPE_LIMIT | TIF_GTC | sell-provider-1|

  And the following trades should be executed:
      | buyer  | price | size | seller  | 
      | network |  50  | 50   | trader3 | 
      | trader2 | 50   | 50   | network | 
      #| network |  10  | 50   | trader2 | 
      #| auxiliary2 | 10   | 50   | network | 

    And the parties should have the following margin levels:
      | party | market id   | maintenance | search | initial | release |
      | trader3 | ETH/DEC19 | 0        | 0   | 0   | 0 |
      | trader2 | ETH/DEC19 | 50       | 75  | 100 | 150 |
     Then the parties should have the following profit and loss:
      | party | volume | unrealised pnl | realised pnl |
      | trader2   | 50 | 2500        | -2400         |
      | trader3   | 0  | 0           | -100          |

      Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | trader2| BTC   | ETH/DEC19 | 200      | 50      |
     
      Then the parties place the following orders:
      | party     | market id | side | volume | price  | resulting trades| type       | tif     | reference |
      | auxiliary2| ETH/DEC19 | buy  | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |
  
      Then the parties should have the following profit and loss:
      | party | volume | unrealised pnl | realised pnl |
      | trader2   | 0 | 0        | -150         |

      And the parties should have the following margin levels:
      | party | market id   | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 0       | 0  | 0 | 0 |

  
#setup new mark price to closeout trader2
    When the parties place the following orders: 
      | party     | market id | side | volume| price | resulting trades | type       | tif     | reference       |
      | auxiliary1| ETH/DEC19 | buy  | 60   | 50    | 0                  | TYPE_LIMIT | TIF_GTC | buy-position-3 |
      | auxiliary2| ETH/DEC19 | sell | 10   | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-position-3 |

    And the parties should have the following margin levels:
      | party | market id   | maintenance | search | initial | release |
      | trader3 | ETH/DEC19 | 0        | 0   | 0   | 0 |
      | trader2 | ETH/DEC19 | 0        | 0   | 0   | 0 |
