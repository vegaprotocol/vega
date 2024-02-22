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
      | party2 | ETH   | 10000  |
      | party3 | ETH   | 10000  |
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


  Scenario: Continuous trading a persistent (GTC) limit order that is not crossed with the order book is included on the order book
            at limit order price at the back of the queue of orders at that price. No trades are generated. (0014-ORDT-081)

    # Place some orders to get some price levels
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party2 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    # Now place the persistent GTC order to rest on the book at the back of the price levels
    # It will not trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy3      |

    # Check that the order was placed at the back of the price level queue by
    # matching against each order on that level in order
    # Match 1
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party5 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTC | 

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party2 | buy2      | STATUS_ACTIVE    |
      | party3 | buy3      | STATUS_ACTIVE    |

    # Match 2
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party5 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTC | 

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party2 | buy2      | STATUS_FILLED    |
      | party3 | buy3      | STATUS_ACTIVE    |

    # Match 3
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party5 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTC | 

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party2 | buy2      | STATUS_FILLED    |
      | party3 | buy3      | STATUS_FILLED    |


  Scenario: Continuous trading a persistent (GTT) limit order that is not crossed with the order book is included on the order book
            at limit order price at the back of the queue of orders at that price. No trades are generated. (0014-ORDT-081)

    # Place some orders to get some price levels
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party2 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    # Now place the persistent GTC order to rest on the book at the back of the price levels
    # It will not trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | party3 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTT | buy3      | 3600       |

    # Check that the order was placed at the back of the price level queue by
    # matching against each order on that level in order
    # Match 1
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party5 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTC | 

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party2 | buy2      | STATUS_ACTIVE    |
      | party3 | buy3      | STATUS_ACTIVE    |

    # Match 2
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party5 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTC | 

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party2 | buy2      | STATUS_FILLED    |
      | party3 | buy3      | STATUS_ACTIVE    |

    # Match 3
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party5 | BTC/ETH   | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTC | 

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party2 | buy2      | STATUS_FILLED    |
      | party3 | buy3      | STATUS_FILLED    |


