Feature: stop orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | spam.protection.max.stopOrdersPerMarket | 5     |

  Scenario: A stop order with reduce only set to false will be rejected. (0014-ORDT-040)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error |
      | party1| ETH/DEC19 | buy  | 1      |  0    | 0                | TYPE_MARKET| TIF_GTC | post| 47                 | stop order must be reduce only   |

  Scenario: A stop order placed by a key with a zero position and no open orders will be rejected. (0014-ORDT-042)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error |
      | party1| ETH/DEC19 | buy  | 1      |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               | stop order submission not allowed without existing position  |

  Scenario: A stop order placed by a key with a zero position but open orders will be accepted. (0014-ORDT-043)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error |
      | party1| ETH/DEC19 | sell | 10     | 60    | 0                | TYPE_LIMIT | TIF_GTC |       |                  |       |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               |       |

  Scenario: Attempting to create more stop orders than is allowed by the relevant network parameter will result in the transaction failing to execute. (0014-ORDT-044)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error |
      | party1| ETH/DEC19 | sell | 10     | 60    | 0                | TYPE_LIMIT | TIF_GTC |       |                  |       |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               |       |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               |       |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               |       |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               |       |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               |       |
      # this next one goes over the top
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_GTC | reduce| 47               | max stop orders per party reached  |

  Scenario: With a last traded price at 50, a stop order placed with Rises Above setting at 75 will be triggered by any trade at price 75 or higher. (0014-ORDT-047)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

   # create party1 stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | ra price trigger | error | reference |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 75               |       | stop1     |

   # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC |


    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 10     | 75    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Rises Above setting at 25 will be triggered immediately (before another trade is even necessary). (0014-ORDT-048)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | ra price trigger | error | reference |
      | party1| ETH/DEC19 | sell | 10     |  0    | 1                | TYPE_MARKET| TIF_IOC | reduce| 25               |       | stop1     |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0     | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Falls Below setting at 25 will be triggered by any trade at price 25 or lower. (0014-ORDT-049)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error | reference |
      | party1| ETH/DEC19 | sell | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 25               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

      # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0     | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Falls Below setting at 75 will be triggered immediately (before another trade is even necessary). (0014-ORDT-050)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error | reference |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 1                | TYPE_MARKET| TIF_IOC | reduce| 75               |       | stop1     |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | STATUS_FILLED | stop1     |

  Scenario: A stop order with expiration time T set to expire at that time will expire at time T if reached without being triggered. (0014-ORDT-052)

    # setup accounts
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    # create party1 stop order, no trade resulting, expires in 10 secs
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | ra price trigger | error | reference | so expires in | so expiry strategy |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 75               |       | stop1     | 10 | EXPIRY_STRATEGY_CANCELS |

    # add 20 secs, should expire
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC19 | STATUS_EXPIRED | stop1     |

  Scenario: A stop order with expiration time T set to execute at that time will execute at time T if reached without being triggered. (0014-ORDT-053)

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

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    When time is updated to "2019-11-30T00:00:10Z"
    # create party1 stop order, no trade resulting, expires in 10 secs
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | ra price trigger | error | reference | so expires in | so expiry strategy |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 75               |       | stop1     | 10 | EXPIRY_STRATEGY_SUBMIT |

    # add 20 secs, should expire
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC19 | STATUS_EXPIRED | stop1     |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | STATUS_FILLED | stop1     |

  Scenario: A stop order set to trade volume x with a trigger set to Rises Above at a given price will trigger at the first trade at or above that price. At this time the order will be placed on the book if and only if it would reduce the trader's absolute position (buying if they are short or selling if they are long) if executed (i.e. will execute as a reduce-only order). (0014-ORDT-055)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | BTC   | 10000000 |
      | aux    | BTC   | 10000000 |
      | aux2   | BTC   | 10000000 |
      | aux3   | BTC   | 10000000 |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then debug trades
    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # before, we check the volum for the party
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error | reference |
      | party1| ETH/DEC19 | sell | 1      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 25               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

      # now we trade at 25, this will breach the trigger
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | STATUS_FILLED | stop1     |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # after the volume has been reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | -30          |

  Scenario: If a pair of stop orders are specified as OCO, one being triggered also removes the other from the book. (0014-ORDT-056)


    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | ra price trigger | error | reference |
      | party1| ETH/DEC19 | sell | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 25               | 75               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

      # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0     | STATUS_FILLED | stop1-1   |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1-1   |
      | party1 | ETH/DEC19 | STATUS_STOPPED   | stop1-2   |

  Scenario: If a pair of stop orders are specified as OCO and one triggers but is invalid at time of triggering (e.g. a buy when the trader is already long) the other will still be cancelled. (0014-ORDT-058)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | BTC   | 10000000 |
      | aux    | BTC   | 10000000 |
      | aux2   | BTC   | 10000000 |
      | aux3   | BTC   | 10000000 |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then debug trades
    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # before, we check the volum for the party
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | ra price trigger |error | reference |
      | party1| ETH/DEC19 | sell | 1      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 25               | 100              |      | stop1     |


    # close party1 position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party1| ETH/DEC19 | sell | 2      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # after the volume has been reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |

    # now we trade at 25, this will breach the trigger
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 1     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 1     | 25    | 1                | TYPE_LIMIT | TIF_GTC |


    # check that the order got submitted and stopped as would not reduce the position
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status         | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | STATUS_STOPPED | stop1-1   |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1-1   |
      | party1 | ETH/DEC19 | STATUS_STOPPED   | stop1-2   |

  Scenario: A trailing stop order for a 5% drop placed when the price is 50, followed by a price rise to 60 will, Be triggered by a fall to 57. (0014-ORDT-059), Not be triggered by a fall to 58. (0014-ORDT-060)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000000|
      | party2 | BTC   | 10000000000|
      | party3 | BTC   | 10000000000|
      | aux    | BTC   | 10000000000|
      | aux2   | BTC   | 10000000000|
      | lpprov | BTC   | 9000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb trailing | error | reference |
      | party1| ETH/DEC19 | sell | 1      |  0    |  0               | TYPE_MARKET| TIF_IOC | reduce| 0.05       |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 20     |  0               | TYPE_LIMIT | TIF_GTC |


    # move prive to 60, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 60    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 10             | 0            |

    # move first to 58, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 58    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 58    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 8             | 0            |

    # move first to 57, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 57    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 57    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  Scenario:  A trailing stop order for a 5% rise placed when the price is 50, followed by a drop to 40 will, Be triggered by a rise to 42. (0014-ORDT-061), Not be triggered by a rise to 41. (0014-ORDT-062)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000000|
      | party2 | BTC   | 10000000000|
      | party3 | BTC   | 10000000000|
      | aux    | BTC   | 10000000000|
      | aux2   | BTC   | 10000000000|
      | lpprov | BTC   | 9000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 short position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | sell  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy   | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | ra trailing | error | reference |
      | party1| ETH/DEC19 | buy  | 1      |  0    |  0               | TYPE_MARKET| TIF_IOC | reduce| 0.05       |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 70     |  0               | TYPE_LIMIT | TIF_GTC |


    # move prive to 60, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 40    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 10             | 0            |

    # move first to 58, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 41    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 41    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 9              | 0            |

    # move first to 57, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 42    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 42    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | buy | 1      | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  Scenario: A trailing stop order for a 25% drop placed when the price is 50, followed by a price rise to 60, then to 50, then another rise to 57 will:, Be triggered by a fall to 45. (0014-ORDT-063), Not be triggered by a fall to 46. (0014-ORDT-064)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000000|
      | party2 | BTC   | 10000000000|
      | party3 | BTC   | 10000000000|
      | aux    | BTC   | 10000000000|
      | aux2   | BTC   | 10000000000|
      | lpprov | BTC   | 9000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb trailing | error | reference |
      | party1| ETH/DEC19 | sell | 1      |  0    |  0               | TYPE_MARKET| TIF_IOC | reduce| 0.25       |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 20     |  0               | TYPE_LIMIT | TIF_GTC |


    # move prive to 60, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 60    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 10             | 0            |

    # move first to 58, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    # move first to 57, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 57    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 57    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 7              | 0            |

    # move first to 46, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 46    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 46    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the volum as not reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | -4              | 0            |


    # move first to 46, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 45    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 45    | 1                | TYPE_LIMIT | TIF_GTC |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  Scenario: A stop order placed either prior to or during an auction will not execute during an auction, nor will it participate in the uncrossing. (0014-ORDT-065)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 100000000    |
      | party2 | BTC   | 100000000    |
      | party3 | BTC   | 100000000    |
      | aux    | BTC   | 100000000   |
      | aux2   | BTC   | 100000000   |
      | lpprov | BTC   | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 51    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 51    | 0                | TYPE_LIMIT | TIF_GTC |
      # setup our order for later
      | party1| ETH/DEC19 | buy  | 1     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order
    # still in auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error | reference |
      | party1| ETH/DEC19 | sell | 1      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 25               |       | stop1     |


    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # trade with party 1
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party2| ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 1     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

    # now we trade at 25, this will breach the trigger
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | sell | 1     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 1     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  Scenario: A stop order placed either prior to or during an auction, where the uncrossing price is within the triggering range, will immediately execute following uncrossing. (0014-ORDT-066)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 100000000    |
      | party2 | BTC   | 100000000    |
      | party3 | BTC   | 100000000    |
      | aux    | BTC   | 100000000   |
      | aux2   | BTC   | 100000000   |
      | lpprov | BTC   | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      # setup our order for later
      | party1| ETH/DEC19 | buy  | 1     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order
    # still in auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error | reference |
      | party1| ETH/DEC19 | sell | 1      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 50               |       | stop1     |


    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

   Then debug orders

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  Scenario: If a trader has open stop orders and their position moves to zero with no open limit orders their stop orders are cancelled. (0014-ORDT-068)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 100000000    |
      | party2 | BTC   | 100000000    |
      | party3 | BTC   | 100000000    |
      | aux    | BTC   | 100000000   |
      | aux2   | BTC   | 100000000   |
      | lpprov | BTC   | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # open position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | buy  | 1     | 50    | 0                | TYPE_LIMIT  | TIF_GTC |
      | party2| ETH/DEC19 | sell | 1     | 50    | 1                | TYPE_LIMIT  | TIF_GTC |

    # create party1 stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only  | fb price trigger | error | reference |
      | party1| ETH/DEC19 | sell | 1      |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce| 25               |       | stop1     |


    # close position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party1| ETH/DEC19 | sell | 1     | 50    | 0                | TYPE_LIMIT  | TIF_GTC |
      | party2| ETH/DEC19 | buy  | 1     | 50    | 1                | TYPE_LIMIT  | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop1     |
