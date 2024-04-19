Feature: stop orders
  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 240               |
      | 600     | 0.99        | 360               |
    And the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.11 | 0.1   | 10          | 11            | 0.1                    |
    Given the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees         | price monitoring    | sla params      |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 240              | default-none | my-price-monitoring | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 240   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
      | spam.protection.max.stopOrdersPerMarket | 5     |

  Scenario: A stop order placed either prior to or during an auction will not execute during an auction, nor will it participate in the uncrossing. (0014-ORDT-147)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |
      | party1 | BTC   | 10000        |
      | party2 | BTC   | 10000        |
      | aux    | BTC   | 100000000000 |
      | aux2   | BTC   | 100000000000 |
      | lpprov | BTC   | 100000000000 |


    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    And time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 0.05        | tstop     |

    # Now we make sure the trailing stop is working correctly
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Now let's move back out of auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And time is updated to "2020-10-16T00:15:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "105" for the market "BTC/ETH"

    # The stop should still be waiting and has not been triggered
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Move the mark price down by <10% to not trigger the stop orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 102   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 102   | 1                | TYPE_LIMIT | TIF_GTC |
    And then the network moves ahead "5" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "102" for the market "BTC/ETH"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Move the mark price down by 10% to trigger the orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 2      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 2      | 95    | 2                | TYPE_LIMIT | TIF_GTC |
    And then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "95" for the market "BTC/ETH"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | tstop     |

  Scenario: A stop order placed prior to an auction will not execute during an auction, nor will it participate in the uncrossing. (0014-ORDT-149)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |
      | party1 | BTC   | 10000        |
      | party2 | BTC   | 10000        |
      | aux    | BTC   | 100000000000 |
      | aux2   | BTC   | 100000000000 |
      | lpprov | BTC   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    And time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 0.05        | tstop     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"

    # Now we make sure the trailing stop is working correctly
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Now let's move back out of auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And time is updated to "2020-10-16T00:15:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "105" for the market "BTC/ETH"

    # The stop should still be waiting and has not been triggered
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Move the mark price down by <10% to not trigger the stop orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | BTC/ETH   | buy  | 1      | 102   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 102   | 1                | TYPE_LIMIT | TIF_GTC |
    And then the network moves ahead "5" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "102" for the market "BTC/ETH"

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Move the mark price down by 10% to trigger the orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 2      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 2      | 95    | 2                | TYPE_LIMIT | TIF_GTC |
    And then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "95" for the market "BTC/ETH"

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | tstop     |

  Scenario: A stop order placed during an auction, where the uncrossing price is within the triggering range, will immediately execute following uncrossing. (0014-ORDT-148)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |
      | party1 | BTC   | 10000        |
      | party2 | BTC   | 10000        |
      | aux    | BTC   | 100000000000 |
      | aux2   | BTC   | 100000000000 |
      | lpprov | BTC   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    And time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | BTC/ETH   | sell | 1      | 85    | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | BTC/ETH   | buy  | 1      | 85    | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.01        | tstop     |

    # Now we make sure the trailing stop is working correctly
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Now let's move back out of auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And time is updated to "2020-10-16T00:15:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "92" for the market "BTC/ETH"

    # The stop should still be waiting and has not been triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | tstop     |

    # check that the order got submitted (it will be stopped due to self trading)
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status         | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1         | 0     | STATUS_STOPPED | tstop     |

  Scenario: A stop order placed prior to an auction, where the uncrossing price is within the triggering range, will immediately execute following uncrossing (0014-ORDT-150)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |
      | party1 | BTC   | 10000        |
      | party2 | BTC   | 10000        |
      | aux    | BTC   | 100000000000 |
      | aux2   | BTC   | 100000000000 |
      | lpprov | BTC   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    And time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.01        | tstop     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | BTC/ETH   | sell | 1      | 85    | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | BTC/ETH   | buy  | 1      | 85    | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"

    # Now we make sure the trailing stop is working correctly
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Now let's move back out of auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And time is updated to "2020-10-16T00:15:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "92" for the market "BTC/ETH"

    # The stop should still be waiting and has not been triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | tstop     |

    # check that the order got submitted (it will be stopped due to self trading)
    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status         | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1         | 0     | STATUS_STOPPED | tstop     |

  Scenario: An order with a stop is placed during continuous trading. The market goes into auction. The market exits auction, the condition for triggering the stop is not met. The stop order is still present. (0014-ORDT-151)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |
      | party1 | BTC   | 10000        |
      | party2 | BTC   | 10000        |
      | aux    | BTC   | 100000000000 |
      | aux2   | BTC   | 100000000000 |
      | lpprov | BTC   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    And time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 0.05        | tstop     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"

    # Now we make sure the trailing stop is working correctly
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # Now let's move back out of auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And time is updated to "2020-10-16T00:15:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "105" for the market "BTC/ETH"

    # The stop should still be waiting and has not been triggered
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

  Scenario: A stop order placed during the opening auction, will be rejected. For spot products (0014-ORDT-153)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference | error                                                   |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 0.05        | tstop     | stop orders are not accepted during the opening auction |

  Scenario: A party places a stop order on a market in continuous trading, the market moves to an auction and the party cancels the stop order. When the market exits the auction the party no longer has a stop order. For spot products (0014-ORDT-152)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |
      | party1 | BTC   | 10000        |
      | party2 | BTC   | 10000        |
      | aux    | BTC   | 100000000000 |
      | aux2   | BTC   | 100000000000 |
      | lpprov | BTC   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/ETH   | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/ETH   | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    And time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference |
      | party1 | BTC/ETH   | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce | 0.01        | tstop     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "BTC/ETH"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | BTC/ETH   | sell | 1      | 85    | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | BTC/ETH   | buy  | 1      | 85    | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"

    # Now we make sure the trailing stop is working correctly
    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | BTC/ETH   | STATUS_PENDING | tstop     |

    # cancel the stop order
    Then the parties cancel the following stop orders:
      | party  | reference |
      | party1 | tstop     |

    # Now let's move back out of auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And time is updated to "2020-10-16T00:15:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "92" for the market "BTC/ETH"

    # The stop should has been cancelled and never triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_CANCELLED | tstop     |
