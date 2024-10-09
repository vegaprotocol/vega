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
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |
      | ETH/DEC20 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | default-basic    | default-eth-for-future | 0.25                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |

  @ZeroPos @Liquidation
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
      | dummy      | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | submission |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lprov | ETH/DEC19 | 100       | 10                   | sell | ASK              | 100    | 55     |
      | lprov | ETH/DEC19 | 100       | 10                   | buy  | BID              | 100    | 55     |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    # trading happens at the end of the open auction period
    Then the parties place the following orders:
      | party      | market id | side | price | volume | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC19 | buy  | 5     | 5      | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC19 | sell | 1000  | 10     | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC19 | buy  | 10    | 10     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC19 | sell | 10    | 10     | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "10" for the market "ETH/DEC19"

    # trader2 posts and order that would take over position of trader3 if they have enough to support it at the new mark price
    When the parties place the following orders:
      | party   | market id | side | price | volume | resulting trades | type       | tif     | reference   |
      | trader2 | ETH/DEC19 | buy  | 50    | 40     | 0                | TYPE_LIMIT | TIF_GTC | buy-order-3 |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 50    | 40     |
      | sell | 1000  | 10     |
      | sell | 1055  | 100    |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general      |
      | trader2 | USD   | ETH/DEC19 | 642    | 1358         |
      | lprov   | USD   | ETH/DEC19 | 7114   | 999999892886 |

# # margin level_trader2= OrderSize*MarkPrice*RF = 40*10*0.801225765=321
# # margin level_Lprov= OrderSize*MarkPrice*RF = 100*10*3.55690359157934000=3557

    # trader3 posts a limit order
    When the parties place the following orders:
      | party   | market id | side | price | volume | resulting trades | type       | tif     | reference       |
      | trader3 | ETH/DEC19 | buy  | 100   | 10     | 0                | TYPE_LIMIT | TIF_GTC | buy-position-31 |
      | dummy   | ETH/DEC19 | sell | 2000  | 1      | 0                | TYPE_LIMIT | TIF_GTC | dummy-sell-1    |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 45    | 100    |
      | buy  | 50    | 40     |
      | buy  | 100   | 10     |
      | sell | 1000  | 10     |
      | sell | 1055  | 100    |
      | sell | 2000  | 1      |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC19 | 81          | 121    | 162     | 243     |
  
    # This should create a zero position, ensure at least 1 tick with the order on the book
    When the network moves ahead "1" blocks
    And the parties cancel the following orders:
      | party | reference    |
      | dummy | dummy-sell-1 |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 45    | 100    |
      | buy  | 50    | 40     |
      | buy  | 100   | 10     |
      | sell | 1000  | 10     |
      | sell | 1055  | 100    |
    #setup for close out
    When the parties place the following orders:
      | party      | market id | side | price | volume | resulting trades | type       | tif     | reference       |
      | auxiliary2 | ETH/DEC19 | sell | 100   | 10     | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
    Then the network moves ahead "3" blocks

    And the mark price should be "100" for the market "ETH/DEC19"

    Then the following trades should be executed:
      | buyer   | price | size | seller     |
      | trader3 | 100   | 10   | auxiliary2 |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 0      |
      | buy  | 45    | 0      |
      | buy  | 50    | 0      |
      | buy  | 100   | 0      |
      | sell | 1000  | 10     |
      | sell | 1055  | 100    |

    #   #trader3 is closed out, trader2 has no more open orders as they got cancelled after becoming distressed
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | trader3 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    # trader3 can not be closed-out because there is not enough vol on the order book
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 0      | 2000    |
      | trader3 | USD   | ETH/DEC19 | 0      | 0       |

    Then the parties should have the following profit and loss:
      | party      | volume | unrealised pnl | realised pnl | status                        | taker fees | taker fees since | maker fees | maker fees since | other fees | other fees since | funding payments | funding payments since |
      | trader2    | 0      | 0              | 0            | POSITION_STATUS_ORDERS_CLOSED | 0          | 0                | 0          | 0                | 0          | 0                | 0                | 0                      |
      | trader3    | 0      | 0              | -162         | POSITION_STATUS_CLOSED_OUT    | 0          | 0                | 0          | 0                | 0          | 0                | 0                | 0                      |
      | auxiliary1 | -10    | -900           | 0            |                               | 0          | 0                | 0          | 0                | 0          | 0                | 0                | 0                      |
      | auxiliary2 | 5      | 475            | 586          |                               | 0          | 0                | 0          | 0                | 1          | 0                | 0                | 0                      |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party      | market id | side | price | volume | resulting trades | type       | tif     | reference       |
      | auxiliary2 | ETH/DEC19 | buy  | 1     | 10     | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |

    When the parties place the following orders:
      | party      | market id | side | price | volume | resulting trades | type       | tif     | reference       |
      | auxiliary2 | ETH/DEC19 | sell | 100   | 10     | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | auxiliary1 | ETH/DEC19 | buy  | 100   | 10     | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |

    Then the network moves ahead "4" blocks
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 1     | 5      |
      | buy  | 5     | 0      |
      | buy  | 45    | 0      |
      | buy  | 50    | 0      |
      | buy  | 100   | 0      |
      | sell | 1000  | 10     |
      | sell | 1055  | 100    |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | trader3 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 0      | 2000    |
      | trader3 | USD   | ETH/DEC19 | 0      | 0       |


  @ZeroMargin
  Scenario: When a party voluntarily closes their position, a zero margin event should be sent
    Given the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 2000          |
      | trader3    | USD   | 162           |
      | lprov      | USD   | 1000000000000 |
      | closer     | USD   | 1000000000000 |
      | dummy      | USD   | 1000000000000 |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | submission |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lprov | ETH/DEC19 | 100       | 10                   | sell | ASK              | 100    | 55     |
      | lprov | ETH/DEC19 | 100       | 10                   | buy  | BID              | 100    | 55     |

    And the parties place the following orders:
      | party      | market id | side | price | volume | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC19 | buy  | 5     | 5      | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC19 | sell | 1000  | 10     | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC19 | buy  | 10    | 10     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC19 | sell | 10    | 10     | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "10" for the market "ETH/DEC19"

    # trader2 posts and order that would take over position of trader3 if they have enough to support it at the new mark price
    When the parties place the following orders:
      | party   | market id | side | price | volume | resulting trades | type       | tif     | reference   |
      | trader2 | ETH/DEC19 | buy  | 50    | 40     | 0                | TYPE_LIMIT | TIF_GTC | buy-order-3 |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 50    | 40     |
      | sell | 1000  | 10     |
      | sell | 1055  | 100    |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general      |
      | trader2 | USD   | ETH/DEC19 | 642    | 1358         |
      | lprov   | USD   | ETH/DEC19 | 7114   | 999999892886 |

    # trader3 posts a limit order
    When the parties place the following orders:
      | party   | market id | side | price | volume | resulting trades | type       | tif     | reference       |
      | trader3 | ETH/DEC19 | buy  | 100   | 10     | 0                | TYPE_LIMIT | TIF_GTC | buy-position-31 |
      | dummy   | ETH/DEC19 | sell | 2000  | 1      | 0                | TYPE_LIMIT | TIF_GTC | dummy-sell-1    |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 45    | 100    |
      | buy  | 50    | 40     |
      | buy  | 100   | 10     |
      | sell | 1000  | 10     |
      | sell | 1055  | 100    |
      | sell | 2000  | 1      |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC19 | 81          | 121    | 162     | 243     |
      | dummy   | ETH/DEC19 | 36          | 54     | 72      | 108     |

    # Party 3 cancels their order
    When the parties cancel the following orders:
      | party | reference    | error |
      | dummy | dummy-sell-1 |       |
    
    # This should cause their margin to be set to 0 thanks to the margin levels event
    Then the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | dummy | ETH/DEC19 | 0           | 0      | 0       | 0       |

