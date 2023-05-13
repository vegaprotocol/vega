Feature: Closeout scenarios
  # This is a test case to demonstrate an order can be rejected when the trader (who places an initial order) does not have enouge collateral to cover the initial margin level

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.55690359157934000
    #risk factor long = 0.801225765
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5           | 2              | 3              |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/DEC20 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | default-basic    | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @EndBlock
  Scenario: 001, 2 parties get close-out at the same time. Distressed position gets taken over by LP, distressed order gets canceled (0005-COLL-002; 0012-POSR-001; 0012-POSR-002; 0012-POSR-004; 0012-POSR-005; 0007-POSN-015)
    # setup accounts, we are trying to closeout trader3 first and then trader2

    Given the insurance pool balance should be "0" for the market "ETH/DEC19"

    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 2000          |
      | trader3    | USD   | 162           |
      | lprov      | USD   | 1000000000000 |
      | closer     | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | buy  | BID              | 100        | 55     | amendment  |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    # trading happens at the end of the open auction period
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC19 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC19 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the auction ends with a traded volume of "10" at a price of "10"
    And the mark price should be "10" for the market "ETH/DEC19"

    # trader2 posts and order that would take over position of trader3 if they have enough to support it at the new mark price
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader2 | ETH/DEC19 | buy  | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-3 |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 1     | 100000 |
      | buy  | 50    | 40     |
      | sell | 1000  | 10     |
      | sell | 1050  | 96     |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search  | initial | release |
      | trader2 | ETH/DEC19 | 321         | 481     | 642     | 963     |
      | lprov   | ETH/DEC19 | 800729      | 1201093 | 1601458 | 2402187 |
    # margin level_trader2= OrderSize*MarkPrice*RF = 40*10*0.801225765=321
    # margin level_Lprov= OrderSize*MarkPrice*RF = max(96*10*3.55690359157934000,100000*10*0.801225765)=801225.765

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 642    | 1358    |

    # trader3 posts a limit order
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | trader3 | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | buy-position-31 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 45    | 2223   |
      | buy  | 50    | 40     |
      | buy  | 100   | 10     |
      | sell | 1000  | 10     |
      | sell | 1055  | 95     |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC19 | 81          | 121    | 162     | 243     |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 642    | 1358    |
      | trader3 | USD   | ETH/DEC19 | 162    | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    Then the parties should have the following profit and loss:
      | party      | volume | unrealised pnl | realised pnl |
      | auxiliary1 | -10    | 0              | 0            |
      | auxiliary2 | 10     | 0              | 0            |
    #setup trader3 position and close it out
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | auxiliary2 | ETH/DEC19 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
    Then the following trades should be executed:
      | buyer      | price | size | seller     |
      | trader3    | 100   | 10   | auxiliary2 |
      | auxiliary2 | 5     | 5    | network    |
      | lprov      | 1     | 5    | network    |
      | network    | 3     | 10   | trader3    |
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 0      |
      | buy  | 1     | 0      |
      | sell | 1000  | 10     |
      | sell | 1005  | 0      |
    #trader3 is closed out, trader2 has no more open orders as they got cancelled after becoming distressed
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | trader3 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 0      | 2000    |
      | trader3 | USD   | ETH/DEC19 | 0      | 0       |

    # 0007-POSN-015
    And the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl | status                        |
      | trader2 | 0      | 0              | 0            | POSITION_STATUS_ORDERS_CLOSED |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the parties should have the following profit and loss:
      | party      | volume | unrealised pnl | realised pnl |
      | auxiliary1 | -10    | -900           | 0            |
      | auxiliary2 | 5      | 475            | 503          |
      | trader2    | 0      | 0              | 0            |
      | trader3    | 0      | 0              | -162         |
      | lprov      | 5      | 495            | -413         |
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                    | auction trigger                            |
      | 100        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS |

  Scenario: 002, Position becomes distressed upon exiting an auction (0007-POSN-016, 0012-POSR-008)
    Given the insurance pool balance should be "0" for the market "ETH/DEC19"
    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 1027          |
      | lprov      | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov | ETH/DEC20 | 100000            | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp1 | lprov | ETH/DEC20 | 100000            | 0.001 | buy  | BID              | 100        | 55     | amendmend  |
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 5       | 10        | 10        | 3556         | 100000         | 10            |
      | 10         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 10      | 10        | 10        | 3556         | 100000         | 10            |

    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary2 | ETH/DEC20 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2    | ETH/DEC20 | sell | 1      | 10    | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC20 | 1026        | 1539   | 2052    | 3078    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |  
      | trader2 | USD   | ETH/DEC20 | 1026   | 0       | 

    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 40    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 29877        | 100000         | 11            |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl | 
      | trader2 | -1     | 0              | 0            | 

    Then the network moves ahead "14" blocks
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 29877        | 100000         | 11            |

    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 40         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 29877        | 100000         | 21            |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl | 
      | trader2 | 0      | 0              | -1026        | 
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 0      | 0       |
      
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC20 | 0           | 0      | 0       | 0       |

    # 0007-POSN-016: The status field will be set to CLOSED_OUT if the party was closed out 

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl | status                    |
      | trader2 | 0      | 0              | -1026        | POSITION_STATUS_CLOSED_OUT|

Scenario: 003, Position becomes distressed when market is in continuous mode (0007-POSN-017)
    Given the insurance pool balance should be "0" for the market "ETH/DEC19"
      Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.01     |
    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 1000          |
      | lprov      | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov | ETH/DEC20 | 400               | 0.001 | sell | ASK              | 100        | 2     | submission |
      | lp1 | lprov | ETH/DEC20 | 400               | 0.001 | buy  | BID              | 100        | 55     | amendmend  |
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC20 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC20 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | open interest |
      | 10         | TRADING_MODE_CONTINUOUS | 1             |

    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary2 | ETH/DEC20 | buy  | 12      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2    | ETH/DEC20 | sell | 12      | 10    | 1                | TYPE_LIMIT | TIF_GTC |
  
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search     | initial    | release    |
      | trader2 | ETH/DEC20 | 1560000427  | 2340000640 | 3120000854 | 4680001281 |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |  
      | trader2 | USD   | ETH/DEC20 | 999    | 0       | 

    # trader2's order (price 1003) has been canceled
    Then the order book should have the following volumes for market "ETH/DEC20":
      | side | price | volume |
      | sell | 1003  | 0      |
      | sell | 1002  | 1      |
      | sell | 1000  | 10     |
      | buy  | 5     | 5      |
      | buy  | 1     | 400    |

    Then the network moves ahead "5" blocks

    # 0007-POSN-017: The status field will be set to DISTRESSED if a trader was distressed based on the margin requirements for their worst possible long/short and they do not have active orders to be closed, however the book currently does not have enough volume to close them out, and will close them out when there is enough volume.
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl | status                    |
      | trader2 | -12    | 0              | 0            | POSITION_STATUS_DISTRESSED|

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | open interest |
      | 10         | TRADING_MODE_CONTINUOUS | 13            |


