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
      | party5 | BTC   | 10000  |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

  Scenario: 0024-OSTA-030,0024-OSTA-031 FOK and IOC order on spot market

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party1 | BTC/ETH   | buy  | 2      | 996   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |       |
      | party1 | BTC/ETH   | buy  | 2      | 998   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |       |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy2      |       |
      | party2 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_FOK | p2-sell1  |       |
      | party2 | BTC/ETH   | sell | 1      | 999   | 0                | TYPE_LIMIT | TIF_FOK | p2-sell2  |       |

    And the orders should have the following status:
      | party  | reference | status         |
      | party2 | p2-sell1  | STATUS_FILLED  |
      | party2 | p2-sell2  | STATUS_STOPPED |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party5 | BTC/ETH   | sell | 5      | 998   | 1                | TYPE_LIMIT | TIF_IOC | p5-sell1  |       |

    And the orders should have the following status:
      | party  | reference | status                  |
      | party5 | p5-sell1  | STATUS_PARTIALLY_FILLED |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party5 | 998   | 2    |

    #the rest of the unfilled IOC order is canceled
    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 998   | 0      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party5 | BTC/ETH   | sell | 5      | 998   | 0                | TYPE_LIMIT | TIF_IOC | p5-sell1  |       |

    #the unfilled IOC order is stopped
    And the orders should have the following status:
      | party  | reference | status         |
      | party5 | p5-sell1  | STATUS_STOPPED |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party5 | BTC/ETH   | sell | 2      | 996   | 1                | TYPE_LIMIT | TIF_IOC | p5-sell1  |       |

    #the filled IOC order is filled
    And the orders should have the following status:
      | party  | reference | status        |
      | party5 | p5-sell1  | STATUS_FILLED |

