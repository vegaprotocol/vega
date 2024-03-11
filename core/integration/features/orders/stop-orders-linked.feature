Feature: linked stop orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3   | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |
      | ETH/DEC20 | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-basic    | default-eth-for-future | 1e-3                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | spam.protection.max.stopOrdersPerMarket | 5     |

    # setup accounts
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 1000000  |
      | party2 | BTC   | 1000000  |
      | party3 | BTC   | 1000000  |
      | aux    | BTC   | 1000000  |
      | aux2   | BTC   | 1000000  |
      | aux3   | BTC   | 1000000  |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 10     |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 10     |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 20     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 20     | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

  

  Scenario: A linked stop order with position size override will be cancelled if the position flips short to long (0014-ORDT-128)

    # party1 will start 10 short
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sellorder |
      | party2| ETH/DEC19 | buy  | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buyorder  |

    # Place a buy position linked stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | reference | ra size override setting | ra size override percentage |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 52               | stop1     | POSITION                 | 1.0                         |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Now let party1 change their position to be long so we can trigger the stop order to be cancelled
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1| ETH/DEC19 | buy  | 11     | 51    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2| ETH/DEC19 | sell | 11     | 51    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should have been cancelled
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop1     |



  Scenario: A linked stop order with position size override will be cancelled if the position flips long to short (0014-ORDT-128)

    # party1 will start 10 long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buyorder  |
      | party2| ETH/DEC19 | sell | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | sellorder |

    # Place a sell position linked stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | fb price trigger | reference | fb size override setting | fb size override percentage |
      | party1| ETH/DEC19 | sell | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 48               | stop1     | POSITION                 | 1.0                         | 

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Now let party1 change their position to be short so we can trigger the stop order to be cancelled
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party2| ETH/DEC19 | buy  | 11     | 49    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party1| ETH/DEC19 | sell | 11     | 49    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should have been cancelled
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop1     |


  Scenario: A linked stop order with position size override will not be cancelled if the position is flat (0014-ORDT-128)

    # party1 will start 10 short
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sellorder |
      | party2| ETH/DEC19 | buy  | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buyorder  |

    # Place a buy position linked stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | reference | ra size override setting | ra size override percentage |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 52               | stop1     | POSITION                 | 1.0                         |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Now let party1 change their position to be flat and check the stop[ order is not cancelled]
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1| ETH/DEC19 | buy  | 10     | 51    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2| ETH/DEC19 | sell | 10     | 51    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should not have triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING   | stop1     |



  Scenario: A linked stop order with position size override will not be cancelled if the position is flat (0014-ORDT-128)

    # party1 will start 10 long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buyorder  |
      | party2| ETH/DEC19 | sell | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | sellorder |

    # Place a sell position linked stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | fb price trigger | reference | fb size override setting | fb size override percentage |
      | party1| ETH/DEC19 | sell | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 48               | stop1     | POSITION                 | 1.0                         |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Now let party1 change their position to be flat and make sure the stop order is not cancelled
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party2| ETH/DEC19 | buy  | 10     | 49    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party1| ETH/DEC19 | sell | 10     | 49    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should not have triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING   | stop1     |



  Scenario: A linked stop order with position size override will flatten the position after being triggered (0014-ORDT-127)

    # party1 will start 10 short
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sellorder |
      | party2| ETH/DEC19 | buy  | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buyorder  |

    # Place a buy position linked stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | reference | ra size override setting | ra size override percentage |
      | party1| ETH/DEC19 | buy  | 2      | 0     | 0                | TYPE_MARKET| TIF_IOC | reduce | 52               | stop1     | POSITION                 | 1.0                         |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Place some orders on the book to give liquidity and to move the last price to trigger the stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party3| ETH/DEC19 | sell | 30     | 52    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2| ETH/DEC19 | buy  | 1      | 52    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should have triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |



  Scenario: A linked stop order with position size override will be flattened when the stop order is triggered (0014-ORDT-127)

    # party1 will start 10 long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buyorder  |
      | party2| ETH/DEC19 | sell | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | sellorder |

    # Place a sell position linked stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | fb price trigger | reference | fb size override setting | fb size override percentage |
      | party1| ETH/DEC19 | sell | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 48               | stop1     | POSITION                 | 1.0                         |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Now let add some liquidity to the book and move the last price to trigger the stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party2| ETH/DEC19 | buy  | 30     | 48    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party3| ETH/DEC19 | sell | 1      | 48    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should have been triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |


  Scenario: A linked stop order with position size override and scaling will partially flatten the position after being triggered (0014-ORDT-129)

    # party1 will start 20 short
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | sell | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sellorder |
      | party2| ETH/DEC19 | buy  | 21     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buyorder  |

    # Place a buy position linked stop order with a size scaling
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | reference | ra size override setting | ra size override percentage |
      | party1| ETH/DEC19 | buy  | 2      | 0     | 0                | TYPE_MARKET| TIF_IOC | reduce | 52               | stop1     | POSITION                 | 0.75                        |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Place some orders on the book to give liquidity and to move the last price to trigger the stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party3| ETH/DEC19 | sell | 30     | 52    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2| ETH/DEC19 | buy  | 1      | 52    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should have triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

    # Check we have created a trade matching the scaled size of the position
    And the following trades should be executed:
      | buyer   | seller  | price  | size |
      | party1  | party3  | 52     | 15    |


  Scenario: A linked stop order with position size override and scaling will partially flattened the position when the stop order is triggered (0014-ORDT-129)

    # party1 will start 20 long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | buy  | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buyorder  |
      | party2| ETH/DEC19 | sell | 21     | 50    | 1                | TYPE_LIMIT | TIF_GTC | sellorder |

    # Place a sell position linked stop order with size scaling
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | fb price trigger | reference | fb size override setting | fb size override percentage |
      | party1| ETH/DEC19 | sell | 2      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 48               | stop1     | POSITION                 | 0.75                        |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # Now let add some liquidity to the book and move the last price to trigger the stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | 
      | party2| ETH/DEC19 | buy  | 30     | 48    | 0                | TYPE_LIMIT | TIF_GTC | 
      | party3| ETH/DEC19 | sell | 1      | 48    | 1                | TYPE_LIMIT | TIF_GTC | 

    # Stop order should have been triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

    # Check we have created a trade matching the scaled size of the position
    And the following trades should be executed:
      | buyer   | seller  | price  | size |
      | party2  | party1  | 48     | 15    |


  Scenario: A linked stop order with position size override and scaling will be validated for correct scaling value

    # party1 will start 10 long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buyorder  |
      | party2| ETH/DEC19 | sell | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | sellorder |

    # Place a stop order with invalid size scaling
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | fb price trigger | reference | fb size override setting | fb size override percentage | error                                                |
      | party1| ETH/DEC19 | sell | 2      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 48               | stop1     | POSITION                 | 0.0                         | stop order size override percentage value is invalid |

    # Place a stop order with invalid size scaling
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | fb price trigger | reference | fb size override setting | fb size override percentage | error                                                |
      | party1| ETH/DEC19 | sell | 2      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 48               | stop1     | POSITION                 | 1.000001                    | stop order size override percentage value is invalid |

    # Place a stop order with invalid size scaling
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | fb price trigger | reference | fb size override setting | fb size override percentage | error                                                |
      | party1| ETH/DEC19 | sell | 2      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 48               | stop1     | POSITION                 | -0.000001                   | stop order size override percentage value is invalid |
