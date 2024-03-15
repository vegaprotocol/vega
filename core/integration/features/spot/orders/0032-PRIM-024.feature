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
      | 3600    | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party1 | BTC   | 1000   |
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

  Scenario: The market continues in regular fashion once price protection auction period ends and price monitoring bounds
            get reset based on last traded price (which may come from the auction itself if it resulted in trades) (0032-PRIM-024)

    # Check that the market price bounds are set 
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

  # Place out of bounds orders to force a price monitoring auction
   Given the parties place the following orders:
    | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
    | party1 | BTC/ETH   | buy  | 1      | 1050  | 0                | TYPE_LIMIT | TIF_GTC | pr-buy    |       |
    | party5 | BTC/ETH   | sell | 1      | 1050  | 0                | TYPE_LIMIT | TIF_GTC | pr-sell   |       |

   When the network moves ahead "1" blocks
   Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

   # Both orders are still alive inside the auction 
   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status        |
    | party1 | BTC/ETH   | pr-buy    | buy  | 1      | 1         | 1050  | STATUS_ACTIVE |
    | party5 | BTC/ETH   | pr-sell   | sell | 1      | 1         | 1050  | STATUS_ACTIVE |

  # Cancel all the orders
   Then the parties cancel the following orders:
    | party  | reference |
    | party1 | pr-buy    |  
    | party5 | pr-sell   |  

  # Place some orders inside the price range that cross
   Given the parties place the following orders:
    | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
    | party1 | BTC/ETH   | buy  | 1      | 1030  | 0                | TYPE_LIMIT | TIF_GTC | pr-buy2   |       |
    | party5 | BTC/ETH   | sell | 1      | 1030  | 0                | TYPE_LIMIT | TIF_GTC | pr-sell2  |       |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Check that the mark price has moved
    And the mark price should be "1030" for the market "BTC/ETH"

   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status        |
    | party1 | BTC/ETH   | pr-buy2   | buy  | 1      | 0         | 1030  | STATUS_FILLED |
    | party5 | BTC/ETH   | pr-sell2  | sell | 1      | 0         | 1030  | STATUS_FILLED |

    # Check that the market price bounds are set to higher levels because the mark price has gone up
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1030       | TRADING_MODE_CONTINUOUS | 3600    | 988       | 1074      | 0            | 0              | 0             |

