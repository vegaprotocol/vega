Feature: Spot market

  Background:

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.value.windowLength               | 1h    |

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
      | 360000  | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party1 | BTC   | 1000   |
      | party2 | ETH   | 100000 |
      | party2 | BTC   | 1000   |
      | party3 | ETH   | 100000 |
      | party5 | BTC   | 1000   |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

  Scenario: 0024-OSTA-033 GTT order on spot market

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | party1 | BTC/ETH   | buy  | 2      | 996   | 0                | TYPE_LIMIT | TIF_GTT | 3          | p1-buy1   |
      | party1 | BTC/ETH   | buy  | 2      | 997   | 0                | TYPE_LIMIT | TIF_GTT | 3          | p1-buy2   |
      | party1 | BTC/ETH   | buy  | 2      | 998   | 0                | TYPE_LIMIT | TIF_GTT | 3          | p1-buy3   |
      | party1 | BTC/ETH   | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTT | 3          | p1-buy4   |
      | party2 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTT | 3          | p2-sell1  |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 1          | p2-sell2  |
    When the network moves ahead "1" blocks

    # GTT order no filled, partially filled, and GTT order expires
    And the orders should have the following status:
      | party  | reference | status         |
      | party1 | p1-buy1   | STATUS_ACTIVE  |
      | party1 | p1-buy2   | STATUS_ACTIVE  |
      | party1 | p1-buy3   | STATUS_ACTIVE  |
      | party2 | p2-sell1  | STATUS_FILLED  |
      | party2 | p2-sell2  | STATUS_EXPIRED |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 999   | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | party2 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTT | p2-sell3  | 2          |
    #GTT order filled
    And the orders should have the following status:
      | party  | reference | status        |
      | party1 | p1-buy4   | STATUS_FILLED |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 999   | 0      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | party2 | BTC/ETH   | sell | 1      | 998   | 1                | TYPE_LIMIT | TIF_GTT | p2-sell3  | 2          |
    #GTCT order partially filled
    And the orders should have the following status:
      | party  | reference | status        |
      | party1 | p1-buy3   | STATUS_ACTIVE |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 998   | 1      |

    Then the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error                                                           |
      | party1 | p1-buy3   | 998   | -1         | TIF_GTT | OrderError: Cannot amend order to GTT without an expiryAt field |

    Then the parties amend the following orders:
      | party  | reference | price | size delta | tif     | expires at | 
      | party1 | p1-buy3   | 998   | -1         | TIF_GTT | 3          | 
#GTT partially filled is canclled by trader
# And the orders should have the following status:
#   | party  | reference | status           |
#   | party1 | p1-buy3   | STATUS_CANCELLED |

# #GTT order rejected by system
# And the parties place the following orders:
#   | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                                              |
#   | party5 | BTC/ETH   | sell | 5000   | 998   | 0                | TYPE_LIMIT | TIF_GTT | p5-sell2  | party does not have sufficient balance to cover the trade and fees |

# And the parties place the following orders:
#   | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
#   | party2 | BTC/ETH   | sell | 1      | 997   | 1                | TYPE_LIMIT | TIF_GTT | p2-sell4  |       |

# #GTT order filled
# And the orders should have the following status:
#   | party  | reference | status        |
#   | party2 | p2-sell4  | STATUS_FILLED |

# And the parties place the following orders:
#   | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
#   | party2 | BTC/ETH   | sell | 2      | 997   | 1                | TYPE_LIMIT | TIF_IOC | p2-sell5  |       |

# #IOC order partially filled and stopped
# And the orders should have the following status:
#   | party  | reference | status                  |
#   | party2 | p2-sell5  | STATUS_PARTIALLY_FILLED |

# And the order book should have the following volumes for market "BTC/ETH":
#   | side | price | volume |
#   | sell | 997   | 0      |

# #IOC order un-filled and stopped
# And the parties place the following orders:
#   | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
#   | party2 | BTC/ETH   | sell | 2      | 997   | 0                | TYPE_LIMIT | TIF_IOC | p2-sell5  |       |

# And the orders should have the following status:
#   | party  | reference | status         |
#   | party2 | p2-sell5  | STATUS_STOPPED |

# And the order book should have the following volumes for market "BTC/ETH":
#   | side | price | volume |
#   | sell | 997   | 0      |
