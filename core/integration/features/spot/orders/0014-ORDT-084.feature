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

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"


  Scenario: Immediate orders, continuous trading Any GTT limit order that [still] resides on the order book at its expiry time is cancelled
            and removed from the book before any events are processed that rely on its being present on the book, including any calculation
            that incorporates its volume and/or price level. (0014-ORDT-084)

    Given time is updated to "2024-01-01T00:01:00Z"
    # Place a GTT order that we are going to expire along with a same price GTC order that will stay around
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | sell1     | 60         |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell2     |            |

    Then the orders should have the following states:
      | party  | market id | reference   | side | volume | remaining | price | status        |
      | party5 | BTC/ETH   | sell1       | sell | 1      | 1         | 1000  | STATUS_ACTIVE |
      | party5 | BTC/ETH   | sell2       | sell | 1      | 1         | 1000  | STATUS_ACTIVE |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1000  | 2      |

    # Move the time forward 1 minute to force the GTT order to expire
    Given time is updated to "2024-01-01T00:02:00Z"

    # Place an order that would have matched with it
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | buy1      |

    # The order should have expired and the volume will be taken from the book
    # resulting in the new buy order only matching with the second sell order that was still active
    And the orders should have the following status:
      | party  | reference | status         |
      | party5 | sell1     | STATUS_EXPIRED |
      | party5 | sell2     | STATUS_FILLED  |
      | party1 | buy1      | STATUS_ACTIVE  |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 1000  | 1      |
