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
      | party4 | BTC   | 1000   |
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
    And the mark price should be "1000" for the market "BTC/ETH"

  Scenario: For a price level with multiple iceberg orders, if an aggressive order hits this price level,
            any volume greater than the displayed volume at a level is split proportionally between the hidden
            components of iceberg orders at that price level If there are three iceberg orders with remaining
            volume 200 lots, 100 lots and 100 lots, an order for 300 lots would be split 150 to the first order
            and 75 to the two 100 lot orders. (0014-ORDT-112)

    # Place multiple iceberg orders to rest on the book
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference |
      | party1 | BTC/ETH   | buy  | 200     | 1000  | 0               | TYPE_LIMIT | TIF_GTC | 5         | 1                    | iceberg1  |
      | party2 | BTC/ETH   | buy  | 100     | 1000  | 0               | TYPE_LIMIT | TIF_GTC | 5         | 1                    | iceberg2  |
      | party1 | BTC/ETH   | buy  | 100     | 1000  | 0               | TYPE_LIMIT | TIF_GTC | 5         | 1                    | iceberg3  |

    # Place an aggressive order to match the icebergs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | BTC/ETH   | sell | 300    | 1000  | 3                | TYPE_LIMIT | TIF_GTC | sell1     |

    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party1 | BTC/ETH   | buy  | 1000  | 5         | 200     | iceberg1  | STATUS_ACTIVE  |             
      | party2 | BTC/ETH   | buy  | 1000  | 5         | 100     | iceberg2  | STATUS_ACTIVE  |             
      | party1 | BTC/ETH   | buy  | 1000  | 5         | 100     | iceberg3  | STATUS_ACTIVE  |             
      | party4 | BTC/ETH   | sell | 1000  | 0         | 300     | sell1     | STATUS_FILLED  | 

    # Check that the orders have been filled with totals of 150, 75 and 75.
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | BTC/ETH   | buy  | 5              | 1000  | STATUS_ACTIVE | 45              |
      | party2 | BTC/ETH   | buy  | 5              | 1000  | STATUS_ACTIVE | 20              |
      | party1 | BTC/ETH   | buy  | 5              | 1000  | STATUS_ACTIVE | 20              |

    # Each of the icebergs have a visible volume of 5 (so 3*5==15 on the book)
    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 1000  | 15     |
      | sell | 1000  | 0      |
