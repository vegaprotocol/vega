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

    Given the log normal risk model named "lognormal-risk-model-2":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.002         | 0.02 | 0  | 0.1 | 1.3   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 30      | 0.999       | 10                |
      | 60      | 0.999       | 20                |
      | 90      | 0.999       | 40                |
      | 120     | 0.999       | 80                |

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

  Scenario: When market is in price monitoring auction, change of a risk model or any of its parameters doesn't affect the previously
            calculated auction end time, any remaining price monitoring bounds cannot extend the auction further. Upon uncrossing
            price monitoring bounds get reset using the updated parameter values. (0032-PRIM-033)

    # Check that the market price bounds are set 
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start | auction end |
      | 1000       | TRADING_MODE_CONTINUOUS | 30      | 997       | 1003      | 0            | 0              | 0             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 995       | 1005      | 0            | 0              | 0             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 90      | 994       | 1006      | 0            | 0              | 0             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 120     | 993       | 1007      | 0            | 0              | 0             | 0             | 0           |

    # Place 2 persistent orders that are outside all of the price bounds
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1008  | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1008  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
    When the network moves ahead "1" blocks

    # Check we have been placed in auction
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # Check we know the auction time
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start       | auction end         |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 995       | 1005      | 0            | 0              | 0             | 1704067203000000000 | 1704067213000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 90      | 994       | 1006      | 0            | 0              | 0             | 1704067203000000000 | 1704067213000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 993       | 1007      | 0            | 0              | 0             | 1704067203000000000 | 1704067213000000000 |

    # Now update the risk model 
    Then the spot markets are updated:
      | id      | risk model             |
      | BTC/ETH | lognormal-risk-model-2 |

    # Make sure the auction time is the same
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start       | auction end         |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 30      | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067213000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067213000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 90      | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067213000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067213000000000 |

    When the network moves ahead "25" blocks

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start       | auction end         |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 30      | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067243000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067243000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 90      | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067243000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 900       | 1100      | 0            | 0              | 0             | 1704067203000000000 | 1704067243000000000 |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # The mark price should show the orders have traded
    And the mark price should be "1007" for the market "BTC/ETH"

   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status         |
    | party1 | BTC/ETH   | buy1      | buy  | 1      | 0         | 1006  | STATUS_FILLED  |
    | party5 | BTC/ETH   | sell1     | sell | 1      | 0         | 1006  | STATUS_FILLED  |

