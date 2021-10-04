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

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
   # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party        | asset | amount        |
      | trader1      | BTC   | 800           |
      | trader2      | BTC   | 4100          |
      | trader3      | BTC   | 99          |
      | auxiliary1   | BTC   | 1000000000000 |
      | auxiliary2   | BTC   | 1000000000000 |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price  | resulting trades| type       | tif     | reference |
      | auxiliary2| ETH/DEC19 | buy  | 10   | 1      | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10   | 1000   | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | auxiliary2| ETH/DEC19 | buy  | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "10" for the market "ETH/DEC19"

# insurance pool generation - setup orderbook and setup position
    When the parties place the following orders:
      | party     | market id | side | volume| price | resulting trades | type       | tif     | reference       |
     # | trader2   | ETH/DEC19 | buy  | 40   | 100    | 0                | TYPE_LIMIT | TIF_GTC | buy-position-2|
      | trader3   | ETH/DEC19 | buy  | 50   | 100    | 0                | TYPE_LIMIT | TIF_GTC | buy-position-3 |
     # | trader1   | ETH/DEC19 | buy  | 40   | 80     | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1 |
      #| trader2   | ETH/DEC19 | buy  | 50   | 90     | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |
      | auxiliary1| ETH/DEC19 | sell | 50   | 100    | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1|
      #| auxiliary2| ETH/DEC19 | buy  | 10     | 100     | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      #| auxiliary1| ETH/DEC19 | sell | 10     | 100     | 1               | TYPE_LIMIT | TIF_GTC | aux-s-1   |

 
    And the mark price should be "100" for the market "ETH/DEC19"

   Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader3 | 50      | 0              | 0        |
      #| trader1 | 0      | 0              | -100         |
      #| trader2 | 0      | 0              | -4010        |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1000  | 10     |
      | buy  | 1     | 10     |

# From spec (https://github.com/vegaprotocol/specs-internal/blob/master/protocol/0019-margin-calculator.md#limit-order-book-linearised-calculation):
# maintenance_margin_long_open_position = max(slippage_volume * slippage_per_unit, 0) + slippage_volume * [ quantitative_model.risk_factors_long ] . [ Product.value(market_observable) ],
#                                       = max(50*99,0)+50*0.1*100 = 4950+500 = 5450

# check whether trader 1/2/3 are all closed out, trader3 first, and forced to transfer 500 position to trader2 for price 9
# then trader2 is closed out, and forced to transfer 40 position to trader 1 for price 8
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      #| trader1 | ETH/DEC19 | 400          | 600     | 800      | 1200      |
      #| trader2 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | trader3 | ETH/DEC19 | 5450          | 8175     | 10900      | 16350      |
      
#     Then the parties should have the following account balances:
#       | party     | asset | market id | margin | general |
#       | trader3  | BTC   | ETH/DEC19 | 0      | 0       |
#      # | trader2   | BTC   | ETH/DEC19 | 0      | 0       |
#      # | trader1   | BTC   | ETH/DEC19 | 0      | 0       |

# #check positions
    Then the parties should have the following profit and loss:
       | party | volume | unrealised pnl | realised pnl |
       | trader3   | 50      | 0           | 0           |



