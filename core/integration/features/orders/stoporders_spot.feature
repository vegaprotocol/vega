Feature: stop orders

  Background:
    Given the spot markets:
      | id       | name     | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | sla params      |
      | BTC/ETH  | BTC/ETH  | BTC        | ETH         | default-simple-risk-model-3   | 1                | default-none | default-none     | default-futures |
      | USDT/ETH | USDT/ETH | USDT       | ETH         | default-log-normal-risk-model | 1                | default-none | default-basic    | default-futures |

    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | spam.protection.max.stopOrdersPerMarket | 5     |

  @NoPerp
  Scenario: If the order is triggered before reaching time T, the order will have been removed and will not trigger at time T. (0014-ORDT-175) (0014-ORDT-124)
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
      | party1 | ETH   | 1000000  |
      | party2 | ETH   | 1000000  |
      | party3 | ETH   | 1000000  |
      | aux    | ETH   | 1000000  |
      | aux2   | ETH   | 1000000  |
      | aux3   | ETH   | 1000000  |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Given time is updated to "2019-11-30T00:00:10Z"
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 20     | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    # volume for the stop trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 20     | 20    | 0                | TYPE_LIMIT | TIF_GTC |
    # create party1 stop order, no trade resulting, expires in 10 secs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference | fb expires in | fb expiry strategy     |
      | party1 | BTC/ETH   | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     | 10            | EXPIRY_STRATEGY_SUBMIT |
    # trigger the stop order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | BTC/ETH   | buy  | 1      | 24    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | BTC/ETH   | sell | 1      | 24    | 1                | TYPE_LIMIT | TIF_GTC |
    # check the stop order is filled
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1     |

    # add 20 secs, should expire
    Given time is updated to "2019-11-30T00:00:30Z"
    # check the stop order was not triggered a second at time T
    # bought 20, sold 10 with the stop order
    Then "party1" should have general account balance of "1000010" for asset "BTC"

  Scenario: Attempting to create more stop orders than is allowed by the relevant network parameter will result in the transaction failing to execute. (0014-ORDT-126)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error                             |
      | party1 | BTC/ETH   | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |                                   |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      # this next one goes over the top
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               | max stop orders per party reached |

  Scenario: An OCO stop order with expiration time T with one side set to execute at that time will execute at time T (0014-ORDT-165)
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | BTC   | 10000000 |
      | aux    | BTC   | 10000000 |
      | aux2   | BTC   | 10000000 |
      | aux3   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |
      | aux    | ETH   | 10000000 |
      | aux2   | ETH   | 10000000 |
      | aux3   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | BTC/ETH   | buy  | 10     | 51    | 0                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      | 10            | EXPIRY_STRATEGY_SUBMIT | 15            | EXPIRY_STRATEGY_SUBMIT |

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | stop-1    |
      | party1 | BTC/ETH   | STATUS_PENDING | stop-2    |

    Then clear all events
    When time is updated to "2019-11-30T00:00:20Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_STOPPED | stop-1    |
      | party1 | BTC/ETH   | STATUS_EXPIRED | stop-2    |

    # Now perform the same test but from the other side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     |
      | party2 | BTC/ETH   | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop2     | 15            | EXPIRY_STRATEGY_SUBMIT | 10            | EXPIRY_STRATEGY_SUBMIT |

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party2 | BTC/ETH   | STATUS_PENDING | stop2-1   |
      | party2 | BTC/ETH   | STATUS_PENDING | stop2-2   |

    Then clear all events
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party2 | BTC/ETH   | STATUS_STOPPED | stop2-2   |
      | party2 | BTC/ETH   | STATUS_EXPIRED | stop2-1   |

  Scenario: If a pair of stop orders are specified as OCO, one being triggered also removes the other from the book. (0014-ORDT-166)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | ra price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               | 75               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | sell | 10     | 0         | 0     | STATUS_FILLED | stop1-1   |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1-1   |
      | party1 | BTC/ETH   | STATUS_STOPPED   | stop1-2   |

  Scenario: A stop order wrapping a limit order will, once triggered, place the limit order as if it just arrived as an order without the stop order wrapping. (0014-ORDT-167)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 1000000  |
      | party2 | BTC   | 1000000  |
      | party3 | BTC   | 1000000  |
      | aux    | BTC   | 1000000  |
      | aux2   | BTC   | 1000000  |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 1000000  |
      | party2 | ETH   | 1000000  |
      | party3 | ETH   | 1000000  |
      | aux    | ETH   | 1000000  |
      | aux2   | ETH   | 1000000  |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 short position
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    # place an order to match with the limit order then check the stop is filled
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 80    | 0                | TYPE_LIMIT | TIF_GTC |
    # create party1 stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | error | reference |
      | party1 | BTC/ETH   | buy  | 5      | 80    | 0                | TYPE_LIMIT | TIF_IOC | reduce | 75               |       | stop1     |

    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 75    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order was triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1     |
    And the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | buy  | 5      | 0         | 80    | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Rises Above setting at 75 will be triggered by any trade at price 75 or higher. (0014-ORDT-168) (0014-ORDT-169)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC |


    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 75    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | buy  | 10     | 0         | 0     | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Rises Above setting at 25 will be triggered immediately (before another trade is even necessary). (0014-ORDT-170)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | sell | 10     | 0         | 0     | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Falls Below setting at 25 will be triggered by any trade at price 25 or lower. (0014-ORDT-171)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | sell | 10     | 0         | 0     | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Falls Below setting at 75 will be triggered immediately (before another trade is even necessary). (0014-ORDT-172)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 5      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | buy  | 10     | 0         | 0     | STATUS_FILLED | stop1     |

  Scenario: A stop order with expiration time T set to expire at that time will expire at time T if reached without being triggered. (0014-ORDT-173)
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    # create party1 stop order, no trade resulting, expires in 10 secs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference | ra expires in | ra expiry strategy      |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     | 10            | EXPIRY_STRATEGY_CANCELS |

    # add 20 secs, should expire
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_EXPIRED | stop1     |

  Scenario: A stop order with expiration time T set to execute at that time will execute at time T if reached without being triggered. (0014-ORDT-174)
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | aux3   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | aux3   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    When time is updated to "2019-11-30T00:00:10Z"
    # create party1 stop order, no trade resulting, expires in 10 secs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference | ra expires in | ra expiry strategy     |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     | 10            | EXPIRY_STRATEGY_SUBMIT |

    # add 20 secs, should expire
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_EXPIRED | stop1     |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | buy  | 10     | 0         | 0     | STATUS_FILLED | stop1     |

  Scenario: A stop order set to trade volume x with a trigger set to Rises Above at a given price will trigger at the first trade at or above that price. At this time the order will be placed on the book if and only if it would reduce the trader's absolute position (buying if they are short or selling if they are long) if executed (i.e. will execute as a reduce-only order). (0014-ORDT-177)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | BTC   | 10000000 |
      | aux    | BTC   | 10000000 |
      | aux2   | BTC   | 10000000 |
      | aux3   | BTC   | 10000000 |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |
      | aux    | ETH   | 10000000 |
      | aux2   | ETH   | 10000000 |
      | aux3   | ETH   | 10000000 |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Then "party1" should have general account balance of "10000001" for asset "BTC"

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

    # now we trade at 25, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | sell | 1      | 0         | 0     | STATUS_FILLED | stop1     |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # after the volume has been reduced
    Then "party1" should have general account balance of "10000000" for asset "BTC"

  Scenario: A trailing stop order for a 5% drop placed when the price is 50, followed by a price rise to 60 will, Be triggered by a fall to 57. (0014-ORDT-141), Not be triggered by a fall to 58. (0014-ORDT-142)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party1 | BTC   | 10000000000 |
      | party2 | BTC   | 10000000000 |
      | party3 | BTC   | 10000000000 |
      | aux    | BTC   | 10000000000 |
      | aux2   | BTC   | 10000000000 |
      | lpprov | BTC   | 9000000000  |
      | party1 | ETH   | 10000000000 |
      | party2 | ETH   | 10000000000 |
      | party3 | ETH   | 10000000000 |
      | aux    | ETH   | 10000000000 |
      | aux2   | ETH   | 10000000000 |
      | lpprov | ETH   | 9000000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | error | reference |
      | party1 | BTC/ETH   | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.05        |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |


    # move prive to 60, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 60    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volum as not reduced
    Then "party1" should have general account balance of "10000000001" for asset "BTC"

    # move first to 58, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 58    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 58    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volum as not reduced
    Then "party1" should have general account balance of "10000000001" for asset "BTC"

    # move first to 57, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 57    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 57    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | sell | 1      | 0         | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1     |

    Then "party1" should have general account balance of "10000000000" for asset "BTC"

  Scenario:  A trailing stop order for a 5% rise placed when the price is 50, followed by a drop to 40 will, Be triggered by a rise to 42. (0014-ORDT-143), Not be triggered by a rise to 41. (0014-ORDT-144)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party1 | BTC   | 10000000000 |
      | party2 | BTC   | 10000000000 |
      | party3 | BTC   | 10000000000 |
      | aux    | BTC   | 10000000000 |
      | aux2   | BTC   | 10000000000 |
      | lpprov | BTC   | 9000000000  |
      | party1 | ETH   | 10000000000 |
      | party2 | ETH   | 10000000000 |
      | party3 | ETH   | 10000000000 |
      | aux    | ETH   | 10000000000 |
      | aux2   | ETH   | 10000000000 |
      | lpprov | ETH   | 9000000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra trailing | error | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.05        |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | sell | 1      | 70    | 0                | TYPE_LIMIT | TIF_GTC |


    # move prive to 60, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 40    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volum as not reduced
    Then "party1" should have general account balance of "9999999999" for asset "BTC"

    # move first to 58, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 41    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 41    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volum as not reduced
    Then "party1" should have general account balance of "9999999999" for asset "BTC"

    # move first to 57, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 42    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 42    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0         | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1     |

    Then "party1" should have general account balance of "10000000000" for asset "BTC"

  Scenario: A trailing stop order for a 25% drop placed when the price is 50, followed by a price rise to 60, then to 50, then another rise to 57 will:, Be triggered by a fall to 45. (0014-ORDT-145), Not be triggered by a fall to 46. (0014-ORDT-146)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party1 | BTC   | 10000000000 |
      | party2 | BTC   | 10000000000 |
      | party3 | BTC   | 10000000000 |
      | aux    | BTC   | 10000000000 |
      | aux2   | BTC   | 10000000000 |
      | lpprov | BTC   | 9000000000  |
      | party1 | ETH   | 10000000000 |
      | party2 | ETH   | 10000000000 |
      | party3 | ETH   | 10000000000 |
      | aux    | ETH   | 10000000000 |
      | aux2   | ETH   | 10000000000 |
      | lpprov | ETH   | 9000000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | error | reference |
      | party1 | BTC/ETH   | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.25        |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |


    # move prive to 60, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 60    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volume has not reduced
    Then "party1" should have general account balance of "10000000001" for asset "BTC"

    # move first to 58, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volum as not reduced
    Then "party1" should have general account balance of "10000000001" for asset "BTC"

    # move first to 57, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 57    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 57    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volum as not reduced
    Then "party1" should have general account balance of "10000000001" for asset "BTC"

    # move first to 46, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 46    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 46    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # check the volum as not reduced
    Then "party1" should have general account balance of "10000000001" for asset "BTC"

    # move first to 46, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 45    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 45    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | sell | 1      | 0         | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1     |

  Scenario: A Stop order that hasn't been triggered can be cancelled. (0014-ORDT-154)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |

    Then the parties cancel the following stop orders:
      | party  | reference |
      | party1 | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop1     |

  @SLABug
  Scenario: All stop orders for a specific party can be cancelled by a single stop order cancellation. (0014-ORDT-155)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 1000000   |
      | party2 | BTC   | 1000000   |
      | aux    | BTC   | 1000000   |
      | aux2   | BTC   | 1000000   |
      | lpprov | BTC   | 900000000 |
      | party1 | ETH   | 1000000   |
      | party2 | ETH   | 1000000   |
      | aux    | ETH   | 1000000   |
      | aux2   | ETH   | 1000000   |
      | lpprov | ETH   | 900000000 |
      | party1 | USDT  | 1000000   |
      | party2 | USDT  | 1000000   |
      | aux    | USDT  | 1000000   |
      | aux2   | USDT  | 1000000   |
      | lpprov | USDT  | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | USDT/ETH  | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USDT/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 48               |       | stop2     |
      | party1 | USDT/ETH  | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop3     |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop4     |

    Then the party "party1" cancels all their stop orders

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop1     |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop2     |
      | party1 | USDT/ETH  | STATUS_CANCELLED | stop3     |
      | party1 | USDT/ETH  | STATUS_CANCELLED | stop4     |

  @SLABug
  Scenario: All stop orders for a specific party for a specific market can be cancelled by a single stop order cancellation. (0014-ORDT-156)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 1000000   |
      | party2 | BTC   | 1000000   |
      | aux    | BTC   | 1000000   |
      | aux2   | BTC   | 1000000   |
      | lpprov | BTC   | 900000000 |
      | party1 | ETH   | 1000000   |
      | party2 | ETH   | 1000000   |
      | aux    | ETH   | 1000000   |
      | aux2   | ETH   | 1000000   |
      | lpprov | ETH   | 900000000 |
      | party1 | USDT  | 1000000   |
      | party2 | USDT  | 1000000   |
      | aux    | USDT  | 1000000   |
      | aux2   | USDT  | 1000000   |
      | lpprov | USDT  | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | USDT/ETH  | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USDT/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 48               |       | stop2     |
      | party1 | USDT/ETH  | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop3     |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop4     |

    Then the party "party1" cancels all their stop orders for the market "BTC/ETH"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop1     |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop2     |
      | party1 | USDT/ETH  | STATUS_PENDING   | stop3     |
      | party1 | USDT/ETH  | STATUS_PENDING   | stop4     |

  Scenario: An OCO stop order with expiration time T with both sides set to execute at that time will be rejected on submission (0014-ORDT-176)
    # setup accounts
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | aux3   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | aux3   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     | error                                              |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop1     | 10            | EXPIRY_STRATEGY_SUBMIT | 10            | EXPIRY_STRATEGY_SUBMIT | stop order OCOs must not have the same expiry time |


  Scenario: An OCO stop order with expiration time T with one side set to execute at that time will execute at time T if reached without being triggered, with the specified side triggering and the other side cancelling. (0014-ORDT-131)
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | BTC   | 10000000 |
      | aux    | BTC   | 10000000 |
      | aux2   | BTC   | 10000000 |
      | aux3   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |
      | aux    | ETH   | 10000000 |
      | aux2   | ETH   | 10000000 |
      | aux3   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | BTC/ETH   | buy  | 10     | 51    | 0                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      | 10            | EXPIRY_STRATEGY_SUBMIT | 15            | EXPIRY_STRATEGY_SUBMIT |

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | stop-1    |
      | party1 | BTC/ETH   | STATUS_PENDING | stop-2    |

    Then clear all events
    When time is updated to "2019-11-30T00:00:20Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_STOPPED | stop-1    |
      | party1 | BTC/ETH   | STATUS_EXPIRED | stop-2    |

    # Now perform the same test but from the other side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     |
      | party2 | BTC/ETH   | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop2     | 15            | EXPIRY_STRATEGY_SUBMIT | 10            | EXPIRY_STRATEGY_SUBMIT |

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party2 | BTC/ETH   | STATUS_PENDING | stop2-1   |
      | party2 | BTC/ETH   | STATUS_PENDING | stop2-2   |

    Then clear all events
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party2 | BTC/ETH   | STATUS_STOPPED | stop2-2   |
      | party2 | BTC/ETH   | STATUS_EXPIRED | stop2-1   |


  Scenario: A stop order placed by a key with a zero position but open orders will be accepted. (0014-ORDT-125)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       |

  Scenario: Given a spot market, a stop order with a position size override will be rejected (0014-ORDT-162)

    # setup accounts
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | BTC   | 10000000 |
      | aux    | BTC   | 10000000 |
      | aux2   | BTC   | 10000000 |
      | aux3   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |
      | aux    | ETH   | 10000000 |
      | aux2   | ETH   | 10000000 |
      | aux3   | ETH   | 100000   |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 5      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     | ra size override setting | ra size override percentage | error                                                      |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      | 10            | EXPIRY_STRATEGY_SUBMIT | 15            | EXPIRY_STRATEGY_SUBMIT | POSITION                 | 0.5                         | stop order size override is not supported for spot product |


  @SLABug
  Scenario: All stop orders for a specific party for a specific market can be cancelled by a single stop order cancellation. (0014-ORDT-156)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 1000000   |
      | party2 | BTC   | 1000000   |
      | aux    | BTC   | 1000000   |
      | aux2   | BTC   | 1000000   |
      | lpprov | BTC   | 900000000 |
      | party1 | ETH   | 1000000   |
      | party2 | ETH   | 1000000   |
      | aux    | ETH   | 1000000   |
      | aux2   | ETH   | 1000000   |
      | lpprov | ETH   | 900000000 |
      | party1 | USDT  | 1000000   |
      | party2 | USDT  | 1000000   |
      | aux    | USDT  | 1000000   |
      | aux2   | USDT  | 1000000   |
      | lpprov | USDT  | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | USDT/ETH  | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USDT/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 48               |       | stop2     |
      | party1 | USDT/ETH  | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop3     |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop4     |

    Then the party "party1" cancels all their stop orders for the market "BTC/ETH"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop1     |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop2     |
      | party1 | USDT/ETH  | STATUS_PENDING   | stop3     |
      | party1 | USDT/ETH  | STATUS_PENDING   | stop4     |

  Scenario: Stop orders once triggered can not be cancelled. For spot products (0014-ORDT-161)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 1000000  |
      | party2 | BTC   | 1000000  |
      | party3 | BTC   | 1000000  |
      | aux    | BTC   | 1000000  |
      | aux2   | BTC   | 1000000  |
      | lpprov | BTC   | 90000000 |
      | party1 | ETH   | 1000000  |
      | party2 | ETH   | 1000000  |
      | party3 | ETH   | 1000000  |
      | aux    | ETH   | 1000000  |
      | aux2   | ETH   | 1000000  |
      | lpprov | ETH   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # setup party1 position, open a 10 short position
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    # place an order to match with the limit order then check the stop is filled
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | sell | 10     | 80    | 0                | TYPE_LIMIT | TIF_GTC |
    # create party1 stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | error      | reference |
      | party1 | BTC/ETH   | buy  | 5      | 80    | 0                | TYPE_LIMIT | TIF_IOC | reduce | 75               |            | stop1     |

    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 75    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order was triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1     |
    And the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | BTC/ETH   | buy  | 5      | 0         | 80    | STATUS_FILLED | stop1     |

    Then the parties cancel the following stop orders:
      | party  | reference | error                |
      | party1 | stop1     | stop order not found |

  @SLABug
  Scenario: All stop orders for a specific party can be cancelled by a single stop order cancellation. (0014-ORDT-155)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 1000000   |
      | party2 | BTC   | 1000000   |
      | aux    | BTC   | 1000000   |
      | aux2   | BTC   | 1000000   |
      | lpprov | BTC   | 900000000 |
      | party1 | ETH   | 1000000   |
      | party2 | ETH   | 1000000   |
      | aux    | ETH   | 1000000   |
      | aux2   | ETH   | 1000000   |
      | lpprov | ETH   | 900000000 |
      | party1 | USDT  | 1000000   |
      | party2 | USDT  | 1000000   |
      | aux    | USDT  | 1000000   |
      | aux2   | USDT  | 1000000   |
      | lpprov | USDT  | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
      | lp2 | lpprov | USDT/ETH  | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | USDT/ETH  | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | USDT/ETH  | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | USDT/ETH  | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USDT/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | BTC/ETH   | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |
      | party1 | BTC/ETH   | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 48               |       | stop2     |
      | party1 | USDT/ETH  | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop3     |
      | party1 | USDT/ETH  | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop4     |

    Then the party "party1" cancels all their stop orders

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop1     |
      | party1 | BTC/ETH   | STATUS_CANCELLED | stop2     |
      | party1 | USDT/ETH  | STATUS_CANCELLED | stop3     |
      | party1 | USDT/ETH  | STATUS_CANCELLED | stop4     |