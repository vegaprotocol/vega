Feature: When market is in price monitoring auction, change of a risk model or any of its parameters doesn't affect the previously calculated
         auction end time, any remaining price monitoring bounds cannot extend the auction further. Upon uncrossing price monitoring bounds get
         reset using the updated parameter values. (0032-PRIM-014)

  Background:
      Given time is updated to "2024-01-01T00:00:00Z"

      Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 30      | 0.999       | 10                |
      | 60      | 0.999       | 20                |
      | 90      | 0.999       | 40                |
      | 120     | 0.999       | 80                |

    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |

    And the log normal risk model named "my-log-normal-risk-model-2":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000002      | 0.00011407711613050421 | 0  | 0.017 | 2.1   |
    
    And the markets:
      | id       | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring      | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | BTC/ETH  | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring   | default-eth-for-future | 0.01                   | 0                         | default-futures |
    
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 60    |
      | limits.markets.maxPeggedOrders          | 2     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.value.windowLength               | 1h    |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party5 | ETH   | 10000000000  |
      | party1 | BTC   | 10000000000  |
      | party5 | BTC   | 10000000000  |

  Scenario: 

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    # Check that the market price bounds are set 
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start | auction end |
      | 1000       | TRADING_MODE_CONTINUOUS | 30      | 994       | 1006      | 74340        | 0              | 1             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 991       | 1009      | 74340        | 0              | 1             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 90      | 989       | 1011      | 74340        | 0              | 1             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 120     | 988       | 1012      | 74340        | 0              | 1             | 0             | 0           |

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
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 991       | 1009      | 149869       | 0              | 1             | 1704067321000000000 | 1704067381000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 90      | 989       | 1011      | 149869       | 0              | 1             | 1704067321000000000 | 1704067381000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 988       | 1012      | 149869       | 0              | 1             | 1704067321000000000 | 1704067381000000000 |

    # Now update the risk model to deactivate all pending price bounds
    Then the markets are updated:
      | id      | risk model                 |
      | BTC/ETH | my-log-normal-risk-model-2 |

    # If we move ahead 25 blocks we should come out of auction instead of it being extended
    When the network moves ahead "25" blocks

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start | auction end |
      | 1008       | TRADING_MODE_CONTINUOUS         | 30      | 1004      | 1012      | 0            | 0              | 0             | 0             | 0           |
      | 1008       | TRADING_MODE_CONTINUOUS         | 60      | 1003      | 1013      | 0            | 0              | 0             | 0             | 0           |
      | 1008       | TRADING_MODE_CONTINUOUS         | 90      | 1001      | 1015      | 0            | 0              | 0             | 0             | 0           |
      | 1008       | TRADING_MODE_CONTINUOUS         | 120     | 1000      | 1016      | 0            | 0              | 0             | 0             | 0           |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # The mark price should show the orders have traded
    And the mark price should be "1008" for the market "BTC/ETH"

   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status         |
    | party1 | BTC/ETH   | buy1      | buy  | 1      | 0         | 1008  | STATUS_FILLED  |
    | party5 | BTC/ETH   | sell1     | sell | 1      | 0         | 1008  | STATUS_FILLED  |

