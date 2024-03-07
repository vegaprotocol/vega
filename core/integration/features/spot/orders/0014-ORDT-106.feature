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

  Scenario: An aggressive iceberg order that crosses with an order where volume < iceberg volume. The initial display quantity is
            filled and the remaining volume is unfilled. Status of iceberg order is active, the volume remaining = (quantity - initial volume)
            and the remaining volume sits on the book. When additional orders are submitted which consume the remaining volume on the iceberg
            order, the volume of the iceberg order is refreshed as and when the volume dips below the minimum peak size (0014-ORDT-106)


    # Add a single order to the book so we can match an iceberg order to it
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | BTC/ETH   | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    # Place an iceberg order to match immediately
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference |
      | party1 | BTC/ETH   | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | 5         | 1                    | iceberg1  |

    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party4 | BTC/ETH   | sell | 1000  | 0         | 5      | sell1     | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 1000  | 5         | 20     | iceberg1  | STATUS_ACTIVE  |             

    # Send in more orders to match with the passive iceberg
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 6      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | sell2     |
    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party5 | BTC/ETH   | sell | 1000  | 0         | 6      | sell2     | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 1000  | 5         | 20     | iceberg1  | STATUS_ACTIVE  |             

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | BTC/ETH   | sell | 3      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | sell3     |
    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party4 | BTC/ETH   | sell | 1000  | 0         | 3      | sell3     | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 1000  | 2         | 20     | iceberg1  | STATUS_ACTIVE  |             

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 6      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | sell4     |
    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party5 | BTC/ETH   | sell | 1000  | 0         | 6      | sell4     | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 1000  | 0         | 20     | iceberg1  | STATUS_FILLED  |           

    # There should be no volume left on the book now
    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 1000  | 0      |
      | sell | 1000  | 0      |


