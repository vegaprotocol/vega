Feature: Spot market

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
      | 360000  | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party3 | BTC   | 100    |
      | party4 | BTC   | 100    |
      | party5 | BTC   | 100    |
    And the average block duration is "1"

  Scenario: A persistent GTC iceberg order that is not crossed with the order book is included in the order book
            with order book volume == initial peak size. No trades are generated (0014-ORDT-087)

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Place some orders to give price levels in the book
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | 5         | 2                    | post |

    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 999   | 1      |
      | buy  | 1000  | 5      |
      | sell | 1001  | 1      |


  Scenario: A persistent GTT iceberg order that is not crossed with the order book is included in the order book
            with order book volume == initial peak size. No trades are generated (0014-ORDT-087)

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Place some orders to give price levels in the book
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | peak size | minimum visible size | only |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 3600       | 5         | 2                    | post |

    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 999   | 1      |
      | buy  | 1000  | 5      |
      | sell | 1001  | 1      |



  Scenario: A persistent GFA iceberg order that is not crossed with the order book is included in the order book
            with order book volume == initial peak size. No trades are generated (0014-ORDT-087)

    # Place some orders to give price levels in the book
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | 5         | 2                    | post |

    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 999   | 1      |
      | buy  | 1000  | 5      |
      | sell | 1001  | 1      |


  Scenario: A persistent GFN iceberg order that is not crossed with the order book is included in the order book
            with order book volume == initial peak size. No trades are generated (0014-ORDT-087)

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Place some orders to give price levels in the book
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFN | 5         | 2                    | post |

    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 999   | 1      |
      | buy  | 1000  | 5      |
      | sell | 1001  | 1      |
