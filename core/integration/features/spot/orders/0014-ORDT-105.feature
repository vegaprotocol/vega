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
      | party4 | BTC   | 100    |
      | party5 | BTC   | 100    |
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

  Scenario: An aggressive iceberg order that crosses with an order where volume > iceberg volume, the iceberg order gets fully filled on entry,
            the iceberg order status is filled, the remaining quantity = 0. Atomic trades are generated if matched against multiple orders (0014-ORDT-105)


    # Place orders on the book ready to match the iceberg orders
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | BTC/ETH   | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
      | party5 | BTC/ETH   | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell2     |


    # Place an iceberg order to match immediately with a volume less than on the book
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference |
      | party1 | BTC/ETH   | buy  | 8      | 1000  | 2                | TYPE_LIMIT | TIF_GTC | 4         | 1                    | iceberg1  |


    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party4 | BTC/ETH   | sell | 1000  | 0         | 5      | sell1     | STATUS_FILLED  | 
      | party5 | BTC/ETH   | sell | 1000  | 2         | 5      | sell2     | STATUS_ACTIVE  |       
      | party1 | BTC/ETH   | buy  | 1000  | 0         | 8      | iceberg1  | STATUS_FILLED  |             