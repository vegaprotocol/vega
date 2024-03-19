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
      | party2 | ETH   | 10000  |
      | party3 | BTC   | 100    |
      | party4 | BTC   | 100    |
      | party5 | BTC   | 100    |
    And the average block duration is "1"

  Scenario: For an iceberg order that's submitted when the market is in auction, iceberg orders trade according to their behaviour if
            they were already on the book (trading first the visible size, then additional if the full visible price level is exhausted
            in the uncrossing) (0014-ORDT-093)


    # Place an iceberg order that we want to full match
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only | reference |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | 2         | 2                    | post | iceberg1  |


    # Place normal GFA orders to match with the full amount of the iceberg order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | iceberg1  | STATUS_FILLED    |
      | party5 | sell1     | STATUS_FILLED    |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party5 |


Scenario: For an iceberg order that's submitted when the market is in auction, iceberg orders trade according to their behaviour if
            they were already on the book (trading first the visible size, then additional if the full visible price level is exhausted
            in the uncrossing) (0014-ORDT-093)


    # Place an iceberg order that we want to full match
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only | reference |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | 2         | 2                    | post | iceberg1  |


    # Place normal GFA orders to partially match with the iceberg order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 8      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | iceberg1  | STATUS_ACTIVE    |
      | party5 | sell1     | STATUS_FILLED    |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 8    | party5 |
