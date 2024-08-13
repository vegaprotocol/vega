Feature: stop orders

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error                                                       |
      | party1 | ETH/DEC19 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               | stop order submission not allowed without existing position |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
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
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error |
      | party1 | ETH/DEC19 | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error                             |
      | party1 | ETH/DEC19 | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |                                   |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |                                   |
      # this next one goes over the top
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               | max stop orders per party reached |

  Scenario: A stop order wrapping a limit order will, once triggered, place the limit order as if it just arrived as an order without the stop order wrapping. (0014-ORDT-045)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 1000000  |
      | party2 | BTC   | 1000000  |
      | party3 | BTC   | 1000000  |
      | aux    | BTC   | 1000000  |
      | aux2   | BTC   | 1000000  |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
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
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    # place an order to match with the limit order then check the stop is filled
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 80    | 0                | TYPE_LIMIT | TIF_GTC |
    # create party1 stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | error | reference |
      | party1 | ETH/DEC19 | buy  | 5      | 80    | 0                | TYPE_LIMIT | TIF_IOC | reduce | 75               |       | stop1     |

    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 10     | 75    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order was triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |
    And the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 5      | 0         | 80    | STATUS_FILLED | stop1     |

  Scenario: With a last traded price at 50, a stop order placed with Rises Above setting at 75 will be triggered by any trade at price 75 or higher. (0014-ORDT-047) (0014-ORDT-046)

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

   # create party1 stop order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     |

   # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC |


    # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 10     | 75    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0         | 0     | STATUS_FILLED | stop1     |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0         | 0     | STATUS_FILLED | stop1     |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

      # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0         | 0     | STATUS_FILLED | stop1     |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 5      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0         | 0     | STATUS_FILLED | stop1     |

  @Liquidation
  Scenario: With a last traded price at 50, a stop order placed with any trigger price which does not trigger immediately will trigger as soon as a trade occurs at a trigger price, and will not wait until the next mark price update to trigger. (0014-ORDT-051)
    # setup network parameters
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
    And the average block duration is "1"

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # place volume to trade with stop order
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |
    # place stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |
    # trigger stop order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 10     | 24    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 10     | 24    | 1                | TYPE_LIMIT | TIF_GTC |
    # check that the stop order was triggered despite the mark price not updating
    Then the mark price should be "50" for the market "ETH/DEC19"
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0         | 0     | STATUS_FILLED | stop1     |

    # check the mark price is later updated correctly
    When the network moves ahead "2" blocks
    Then the mark price should be "20" for the market "ETH/DEC19"

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    # create party1 stop order, no trade resulting, expires in 10 secs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference | ra expires in | ra expiry strategy      |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     | 10            | EXPIRY_STRATEGY_CANCELS |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    When time is updated to "2019-11-30T00:00:10Z"
    # create party1 stop order, no trade resulting, expires in 10 secs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference | ra expires in | ra expiry strategy     |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               |       | stop1     | 10            | EXPIRY_STRATEGY_SUBMIT |

    # add 20 secs, should expire
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC19 | STATUS_EXPIRED | stop1     |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 10     | 0         | 0     | STATUS_FILLED | stop1     |

  Scenario: If the order is triggered before reaching time T, the order will have been removed and will not trigger at time T. (0014-ORDT-054) (0014-ORDT-041)

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
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given time is updated to "2019-11-30T00:00:10Z"
    # setup party1 position, open a 10 long position
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 20     | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    # volume for the stop trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 20     | 20    | 0                | TYPE_LIMIT | TIF_GTC |
    # create party1 stop order, no trade resulting, expires in 10 secs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference | fb expires in | fb expiry strategy     |
      | party1 | ETH/DEC19 | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     | 10            | EXPIRY_STRATEGY_SUBMIT |

    # trigger the stop order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 1      | 24    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 1      | 24    | 1                | TYPE_LIMIT | TIF_GTC |
    # check the stop order is filled
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

    # add 20 secs, should expire
    Given time is updated to "2019-11-30T00:00:30Z"
    # check the stop order was not triggered a second at time T
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | -300           | -300         |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |


    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # before, we check the volum for the party
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

      # now we trade at 25, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0         | 0     | STATUS_FILLED | stop1     |

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |


    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | ra price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               | 75               |       | stop1     |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 10     | 20    | 0                | TYPE_LIMIT | TIF_GTC |

      # now we trade at 75, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 25    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 10     | 0         | 0     | STATUS_FILLED | stop1-1   |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1-1   |
      | party1 | ETH/DEC19 | STATUS_STOPPED   | stop1-2   |

  @Liquidation
  Scenario: If a pair of stop orders are specified as OCO and one triggers but is invalid at time of triggering (e.g. a buy when the trader is already long) the other will still be cancelled. (0014-ORDT-058)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 10000000  |
      | party2 | BTC   | 10000000  |
      | party3 | BTC   | 900000000 |
      | aux    | BTC   | 10000000  |
      | aux2   | BTC   | 10000000  |
      | aux3   | BTC   | 10000000  |
      | lpprov | BTC   | 90000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |


    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # before, we check the volum for the party
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | ra price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               | 100              |       | stop1     |


    # close party1 position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | sell | 2      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # after the volume has been reduced
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |

    # now we trade at 25, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | sell | 1      | 25    | 0                | TYPE_LIMIT | TIF_GTC | p3-ord    |
      | party2 | ETH/DEC19 | buy  | 1      | 25    | 1                | TYPE_LIMIT | TIF_GTC | p2-ord    |


    # check that the order got submitted and stopped as would not reduce the position
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status         | reference |
      | party1 | ETH/DEC19 | sell | 1      | 1         | 0     | STATUS_STOPPED | stop1-1   |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1-1   |
      | party1 | ETH/DEC19 | STATUS_STOPPED   | stop1-2   |

  Scenario: A trailing stop order for a 5% drop placed when the price is 50, followed by a price rise to 60 will, Be triggered by a fall to 57. (0014-ORDT-059), Not be triggered by a fall to 58. (0014-ORDT-060)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party1 | BTC   | 10000000000 |
      | party2 | BTC   | 10000000000 |
      | party3 | BTC   | 10000000000 |
      | aux    | BTC   | 10000000000 |
      | aux2   | BTC   | 10000000000 |
      | lpprov | BTC   | 9000000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | error | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.05        |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |


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
      | party1 | 1      | 8              | 0            |

    # move first to 57, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 57    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 57    | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0         | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  Scenario:  A trailing stop order for a 5% rise placed when the price is 50, followed by a drop to 40 will, Be triggered by a rise to 42. (0014-ORDT-061), Not be triggered by a rise to 41. (0014-ORDT-062)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party1 | BTC   | 10000000000 |
      | party2 | BTC   | 10000000000 |
      | party3 | BTC   | 10000000000 |
      | aux    | BTC   | 10000000000 |
      | aux2   | BTC   | 10000000000 |
      | lpprov | BTC   | 9000000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra trailing | error | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.05        |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 70    | 0                | TYPE_LIMIT | TIF_GTC |


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
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 0         | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  Scenario: A trailing stop order for a 25% drop placed when the price is 50, followed by a price rise to 60, then to 50, then another rise to 57 will:, Be triggered by a fall to 45. (0014-ORDT-063), Not be triggered by a fall to 46. (0014-ORDT-064)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party1 | BTC   | 10000000000 |
      | party2 | BTC   | 10000000000 |
      | party3 | BTC   | 10000000000 |
      | aux    | BTC   | 10000000000 |
      | aux2   | BTC   | 10000000000 |
      | lpprov | BTC   | 9000000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order, results in a trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | error | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.25        |       | stop1     |

    # create volume to close party 1
    # high price sell so it doesn't trade
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |


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
      | party1 | 1      | -4             | 0            |


    # move first to 46, nothing happen
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 45    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 45    | 1                | TYPE_LIMIT | TIF_GTC |


    # check that the order got submitted
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0         | 0     | STATUS_FILLED | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |

  
  Scenario: If a trader has open stop orders and their position moves to zero whilst they still have open limit orders their stop orders will remain active. (0014-ORDT-067)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | aux    | BTC   | 100000000 |
      | aux2   | BTC   | 100000000 |
      | lpprov | BTC   | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
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
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    # create party1 stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |
    # create party1 limit orders
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 51    | 0                | TYPE_LIMIT | TIF_GTC |

    # close position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks

    # check stop orders have not been cancelled and are still pending
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING | stop1     |

  Scenario: If a trader has open stop orders and their position moves to zero with no open limit orders their stop orders are cancelled. (0014-ORDT-068)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | aux    | BTC   | 100000000 |
      | aux2   | BTC   | 100000000 |
      | lpprov | BTC   | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # create party1 stop order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               |       | stop1     |


    # close position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop1     |


  Scenario: A Stop order that hasn't been triggered can be cancelled. (0014-ORDT-071)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

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
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |

    Then the parties cancel the following stop orders:
      | party  | reference |
      | party1 | stop1     |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop1     |

  @SLABug
  Scenario: All stop orders for a specific party can be cancelled by a single stop order cancellation. (0014-ORDT-072)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 1000000   |
      | party2 | BTC   | 1000000   |
      | aux    | BTC   | 1000000   |
      | aux2   | BTC   | 1000000   |
      | lpprov | BTC   | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC20 | 900000            | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC20 | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | ETH/DEC19 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |
      | party1 | ETH/DEC19 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 48               |       | stop2     |
      | party1 | ETH/DEC20 | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | ETH/DEC20 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop3     |
      | party1 | ETH/DEC20 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop4     |

    Then the party "party1" cancels all their stop orders

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop1     |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop2     |
      | party1 | ETH/DEC20 | STATUS_CANCELLED | stop3     |
      | party1 | ETH/DEC20 | STATUS_CANCELLED | stop4     |

  @SLABug
  Scenario: All stop orders for a specific party for a specific market can be cancelled by a single stop order cancellation. (0014-ORDT-073)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 1000000   |
      | party2 | BTC   | 1000000   |
      | aux    | BTC   | 1000000   |
      | aux2   | BTC   | 1000000   |
      | lpprov | BTC   | 900000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC20 | 900000            | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC20 | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | error | reference |
      | party1 | ETH/DEC19 | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | ETH/DEC19 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 47               |       | stop1     |
      | party1 | ETH/DEC19 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 48               |       | stop2     |
      | party1 | ETH/DEC20 | sell | 10     | 60    | 0                | TYPE_LIMIT  | TIF_GTC |        |                  |       |           |
      | party1 | ETH/DEC20 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop3     |
      | party1 | ETH/DEC20 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 49               |       | stop4     |

    Then the party "party1" cancels all their stop orders for the market "ETH/DEC19"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop1     |
      | party1 | ETH/DEC19 | STATUS_CANCELLED | stop2     |
      | party1 | ETH/DEC20 | STATUS_PENDING   | stop3     |
      | party1 | ETH/DEC20 | STATUS_PENDING   | stop4     |

  Scenario: A stop order cannot be triggered by orders crossing during an auction. (WIP TEST CASE 2)

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | BTC   | 1000000000 |
      | party2 | BTC   | 1000000000 |
      | party3 | BTC   | 1000000000 |
      | aux    | BTC   | 1000000000 |
      | aux2   | BTC   | 1000000000 |
      | lpprov | BTC   | 9000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp2 | lpprov | ETH/DEC20 | 900000            | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC20 | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 100    |
    
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 5000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 5       | 4993      | 5007      |
      | 5000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 10      | 4986      | 5014      |

    # Open a position for party1
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC |
    # Place a stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | reference |
      | party1 | ETH/DEC20 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 5001             |       | stop      |
    # Trigger a price-monitoring auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC20 | buy  | 1      | 5010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC20 | sell | 1      | 5010  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | horizon | min bound | max bound |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 10      | 4986      | 5014      |
    # Check the stop order was not triggered
    And the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC20 | STATUS_PENDING | stop      |

    # Update the indicative uncrossing price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC20 | buy  | 1      | 5011  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC20 | sell | 1      | 5011  | 0                | TYPE_LIMIT | TIF_GTC |
    # Check the stop order was not triggered
    And the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC20 | STATUS_PENDING | stop      |

  Scenario: A stop order cannot be triggered by a stop order expiring during an auction. (WIP TEST CASE 2)

    # setup accounts
    Given time is updated to "2019-11-30T00:00:00Z"
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 100    |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 5      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 5      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 5000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 5       | 4993      | 5007      |
      | 5000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 10      | 4986      | 5014      |

    # Open a position for party1
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC |
    # Place a stop order which will expire during the auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error | ra expires in | ra expiry strategy     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 5020             |       | 5             | EXPIRY_STRATEGY_SUBMIT | stop      |
    # Trigger a price-monitoring auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC20 | buy  | 1      | 5010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC20 | sell | 1      | 5010  | 0                | TYPE_LIMIT | TIF_GTC |
    # Check we have entered an auction
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | horizon | min bound | max bound |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 10      | 4986      | 5014      |
    # Check the stop order was not triggered
    And the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC20 | STATUS_PENDING | stop      |

    # Update the time to expire the stop order
    When time is updated to "2019-11-30T00:00:03Z"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | horizon | min bound | max bound |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 10      | 4986      | 5014      |
    And the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC20 | STATUS_PENDING | stop      |

    # Update the time to the end of the auction
    When time is updated to "2019-11-30T00:00:10Z"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 5010       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 10      | 4996      | 5024      |
    And the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC20 | STATUS_EXPIRED | stop      |
    # The stop order did not trigger an order as stop expired during an auction
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 10             | 0            |


  Scenario: If the order is triggered before reaching time T, the order will have been removed and will not trigger at time T. (0014-ORDT-054) (0014-ORDT-041)

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
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given time is updated to "2019-11-30T00:00:10Z"
    # setup party1 position, open a 10 long position
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 20     | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    # volume for the stop trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 20     | 20    | 0                | TYPE_LIMIT | TIF_GTC |
    # create party1 stop order, no trade resulting, expires in 10 secs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb price trigger | ra price trigger | error | reference | fb expires in | fb expiry strategy     |
      | party1 | ETH/DEC19 | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 25               | 100              |       | stop1     | 10            | EXPIRY_STRATEGY_SUBMIT |

    # trigger the stop order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 1      | 24    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 1      | 24    | 1                | TYPE_LIMIT | TIF_GTC |
    # check the stop order is filled
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1-1   |
      | party1 | ETH/DEC19 | STATUS_STOPPED   | stop1-2   |

    # add 20 secs, should expire
    Given time is updated to "2019-11-30T00:00:30Z"
    # check the stop order was not triggered a second at time T
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | -300           | -300         |


  Scenario: A stop order expiring in the past is rejected

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    When time is updated to "2019-11-30T00:00:10Z"
    # create party1 stop order, no trade resulting, expires in 10 secs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | error                         | reference | ra expires in | ra expiry strategy     |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | stop order expiry in the past | stop1     | -10           | EXPIRY_STRATEGY_SUBMIT |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_REJECTED | stop1     |



  Scenario: An OCO stop order with expiration time T with both sides set to execute at that time will be rejected on submission (0014-ORDT-130)

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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |


    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     | error                                              |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop1     | 10            | EXPIRY_STRATEGY_SUBMIT | 10            | EXPIRY_STRATEGY_SUBMIT | stop order OCOs must not have the same expiry time |


  Scenario: An OCO stop order with expiration time T with one side set to execute at that time will execute at time T 
          # if reached without being triggered, with the specified side triggering and the other side cancelling. (0014-ORDT-131)

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

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 10     | 51    | 0                | TYPE_LIMIT | TIF_GTC |


    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     | 
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      | 10            | EXPIRY_STRATEGY_SUBMIT | 15            | EXPIRY_STRATEGY_SUBMIT | 

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop-1    |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop-2    |

    Then clear all events
    When time is updated to "2019-11-30T00:00:20Z"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_STOPPED   | stop-1    |
      | party1 | ETH/DEC19 | STATUS_EXPIRED   | stop-2    |

    # Now perform the same test but from the other side 
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     | 
      | party2 | ETH/DEC19 | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop2     | 15            | EXPIRY_STRATEGY_SUBMIT | 10            | EXPIRY_STRATEGY_SUBMIT | 

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party2 | ETH/DEC19 | STATUS_PENDING  | stop2-1   |
      | party2 | ETH/DEC19 | STATUS_PENDING  | stop2-2   |

    Then clear all events
    When time is updated to "2019-11-30T00:00:30Z"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party2 | ETH/DEC19 | STATUS_STOPPED   | stop2-2   |
      | party2 | ETH/DEC19 | STATUS_EXPIRED   | stop2-1   |


  Scenario: A party with a long or short position CAN increase their position with stop orders. (0014-ORDT-137)

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

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 10     | 51    | 0                | TYPE_LIMIT | TIF_GTC |

    # We should not be able to place a but stop order for party2 as they have a long position and it would make it more long
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | error |
      | party2 | ETH/DEC19 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      |       |

    # We should not be able to place a sell stop order for party1 as they have a short position and it would make it more short
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | error |
      | party1 | ETH/DEC19 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      |       |

  Scenario: A party with a long position cannot flip to short by placing a stop order.
    Given time is updated to "2019-11-30T00:00:00Z"
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | BTC   | 10000000 |
      | aux    | BTC   | 10000000 |
      | aux2   | BTC   | 10000000 |
      | aux3   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC19"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party2 | party1 | 50    | 10   |
    
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 75    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 50     | 75    | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party1 | 75    | 1    |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | error |
      | party2 | ETH/DEC19 | sell | 20     | 0     | 1                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      |       |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party2 | 75    | 10   |

    # Ensure the party has closed its position, despite the stop order being for a larger volume than their open position.
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -11    | -250           | 0            |
      | party2 | 0      | 0              | 250          |
      | party3 | 11     | 0              | 0            |

  Scenario: If a stop order is placed with a position_fraction equal to 0.5 and the position
            size is 5 then the rounding should be equal to 3 (0014-ORDT-138)

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

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 5      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 10     | 51    | 0                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | fb price trigger | reference | ra expires in | ra expiry strategy     | fb expires in | fb expiry strategy     | ra size override setting       | ra size override percentage |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | 25               | stop      | 10            | EXPIRY_STRATEGY_SUBMIT | 15            | EXPIRY_STRATEGY_SUBMIT | POSITION                       | 0.5                         |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop-1    |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop-2    |

    Then clear all events
    When time is updated to "2019-11-30T00:00:20Z"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_STOPPED   | stop-1    |
      | party1 | ETH/DEC19 | STATUS_EXPIRED   | stop-2    |

    # Now we check that the order size was 3 as the position was 5 and the were scaling by 0.5 (5*0.5)==3, we round up
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 50    | 3    | party3 |


  Scenario: If a stop order is placed with a position_fraction equal to 0 the order should be rejected. (0014-ORDT-139)

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

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 100    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 5      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # volume for the stop trade
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 10     | 51    | 0                | TYPE_LIMIT | TIF_GTC |

    When time is updated to "2019-11-30T00:00:10Z"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | ra price trigger | reference | ra expires in | ra expiry strategy     | ra size override setting       | ra size override percentage | error                                                |
      | party1 | ETH/DEC19 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 75               | stop      | 10            | EXPIRY_STRATEGY_SUBMIT | POSITION                       | 0.0                         | stop order size override percentage value is invalid |


