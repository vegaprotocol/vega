Feature: Spot market

  Background:

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


  Scenario: Immediate orders, continuous trading an aggressive persistent (GTC) limit order that crosses with trades >= to its volume
            is filled completely and does not appear on the order book or in the order book volume. Trades are atomically generated for the
            full volume. (0014-ORDT-082)

    # Place some orders to get some volume at a fixed price level
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell3     |
      | party4 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell4     |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell5     |

    # Now place the aggressive GTC order to fully fill from the available volume on the book
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 3      | 1000  | 3                | TYPE_LIMIT | TIF_GTC | buy1      |

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party3 | sell3     | STATUS_FILLED    |
      | party4 | sell4     | STATUS_FILLED    |
      | party5 | sell5     | STATUS_FILLED    |


  Scenario: Immediate orders, continuous trading an aggressive persistent (GTT) limit order that crosses with trades >= to its volume
            is filled completely and does not appear on the order book or in the order book volume. Trades are atomically generated for the
            full volume. (0014-ORDT-082)

    # Place some orders to get some volume at a fixed price level
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell3     |
      | party4 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell4     |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell5     |

    # Now place the aggressive GTC order to fully fill from the available volume on the book
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | party1 | BTC/ETH   | buy  | 3      | 1000  | 3                | TYPE_LIMIT | TIF_GTT | buy1      | 3600       |

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party3 | sell3     | STATUS_FILLED    |
      | party4 | sell4     | STATUS_FILLED    |
      | party5 | sell5     | STATUS_FILLED    |

