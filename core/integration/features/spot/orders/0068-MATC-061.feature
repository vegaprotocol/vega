Feature: Spot market order tests. Covers 0068-MATC-061, 0068-MATC-062, 0068-MATC-063, 0068-MATC-064, 0068-MATC-065.

  Background:
    Given time is updated to "2024-01-01T00:00:00Z"

    Given the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.value.windowLength                           | 1h    |
    
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.999       | 10                |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party1 | BTC   | 1000   |
      | party2 | ETH   | 10000  |
      | party4 | BTC   | 1000   |
      | party5 | BTC   | 1000   |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 998   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1050  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

  Scenario: 0068-MATC-061: Incoming MARKET orders will be matched against the opposite side of the book.

    # set up the book with some volume the market order will uncross with
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | BTC/ETH   | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p1-ioc-pas |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference  |
      | party5 | BTC/ETH   | sell | 2      | 0     | 1                | TYPE_MARKET | TIF_IOC | p5-ioc-agg |
	Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 2    | party5 |
    And the orders should have the following status:
      | party  | reference  | status        |
      | party1 | p1-ioc-pas | STATUS_ACTIVE |
      | party5 | p5-ioc-agg | STATUS_FILLED |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

  Scenario: 0068-MATC-062: If not enough volume is available to fully fill the order, the remaining will be cancelled.
    # set up the book with some volume the market order will partially uncross with
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | BTC/ETH   | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p1-ioc-pas |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference  |
      | party5 | BTC/ETH   | sell | 5      | 0     | 2                | TYPE_MARKET | TIF_IOC | p5-ioc-agg |
	Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 2    | party5 |
      | party1 | 998   | 1    | party5 |
    # filled the order of party 1, the remainder of p5's order is marked as partially filled.
    And the orders should have the following status:
      | party  | reference  | status                  |
      | party1 | p1-ioc-pas | STATUS_FILLED           |
      | party5 | p5-ioc-agg | STATUS_PARTIALLY_FILLED |

    # To ensure the order is cancelled (meaning in a final state) let's ensure we can't amend the order:
    When the parties amend the following orders:
      | party  | reference  | tif     | size delta | error                        |
      | party5 | p5-ioc-agg | TIF_IOC | -1         | OrderError: Invalid Order ID |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 998        | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

  Scenario: 0068-MATC-064: If there is no match the order will be cancelled. (0068-MATC-063: Incoming LIMIT orders will be matched against the opposite side of the book)
    # set up the book with some volume the market order will uncross with
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party5 | BTC/ETH   | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_IOC | p5-ioc-agg |
    Then the orders should have the following status:
      | party  | reference  | status         |
      | party5 | p5-ioc-agg | STATUS_STOPPED |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

  Scenario: 0068-MATC-065: If there is a partial match then the remaining will be cancelled. (0068-MATC-063: Incoming LIMIT orders will be matched against the opposite side of the book)
    # set up the book with some volume the limit order will partially uncross with
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | BTC/ETH   | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p1-ioc-pas |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party5 | BTC/ETH   | sell | 5      | 1000  | 1                | TYPE_LIMIT | TIF_IOC | p5-ioc-agg |
	Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 2    | party5 |
    # filled the order of party 1, the remainder of p5's order is marked as partially filled.
    And the orders should have the following status:
      | party  | reference  | status                  |
      | party1 | p1-ioc-pas | STATUS_FILLED           |
      | party5 | p5-ioc-agg | STATUS_PARTIALLY_FILLED |

    # To ensure the order is cancelled (meaning in a final state) let's ensure we can't amend the order:
    When the parties amend the following orders:
      | party  | reference  | tif     | size delta | error                        |
      | party5 | p5-ioc-agg | TIF_IOC | -1         | OrderError: Invalid Order ID |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |
