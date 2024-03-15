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

  Scenario: Non-persistent order does not result in an auction (1 out of 2 triggers breached),
            order gets cancelled (never makes it to the order book) (0032-PRIM-023)

    # Check that the market price bounds are set 
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

  # Try to place an IOC order at a price that would trigger a price monitoring auction
   Given the parties place the following orders:
    | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
    | party1 | BTC/ETH   | buy  | 1      | 1050  | 0                | TYPE_LIMIT | TIF_GTC | pr-buy    |       |
    | party5 | BTC/ETH   | sell | 1      | 1050  | 0                | TYPE_LIMIT | TIF_IOC | pr-sell   | OrderError: non-persistent order trades out of price bounds |

   When the network moves ahead "1" blocks
   Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status         |
    | party5 | BTC/ETH   | pr-sell   | sell | 1      | 1         | 1050  | STATUS_STOPPED |

  # Try to place an FOK order at a price that would trigger a price monitoring auction
   Given the parties place the following orders:
    | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
    | party5 | BTC/ETH   | sell | 1      | 1050  | 0                | TYPE_LIMIT | TIF_FOK | pr-sell2  | OrderError: non-persistent order trades out of price bounds |

   When the network moves ahead "1" blocks
   Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status         |
    | party5 | BTC/ETH   | pr-sell2  | sell | 1      | 1         | 1050  | STATUS_STOPPED |


