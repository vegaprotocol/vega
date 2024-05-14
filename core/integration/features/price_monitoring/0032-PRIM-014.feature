Feature: When market is in price monitoring auction, change of a risk model or any of its parameters doesn't affect the previously calculated
         auction end time, any remaining price monitoring bounds cannot extend the auction further. Upon uncrossing price monitoring bounds get
         reset using the updated parameter values. (0032-PRIM-014)

  Background:
      Given time is updated to "2024-01-01T00:00:00Z"

      Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |

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
      | 0.000001      | 0.00011407711613050421 | 0  | 0.016 | 2.0   |
    
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring      | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC24 | ETH        | ETH   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring   | default-eth-for-future | 0.01                   | 0                         | default-futures |
    
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 10    |
      | limits.markets.maxPeggedOrders          | 2     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.value.windowLength               | 1h    |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party5 | ETH   | 10000000000  |
      | lp1    | ETH   | 10000000000  |
      | lp2    | ETH   | 10000000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lpp1 | lp1 | ETH/DEC24 | 90000000          | 0.1 | submission |
      | lpp2 | lp2 | ETH/DEC24 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1 | ETH/DEC24 | 2         | 1                    | buy  | BID              | 50         | 500    |
      | lp2 | ETH/DEC24 | 2         | 1                    | sell | ASK              | 50         | 500    |

  Scenario: 

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC24 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | ETH/DEC24 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "ETH/DEC24"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC24"
    And the mark price should be "1000" for the market "ETH/DEC24"

    # Check that the market price bounds are set 
    And the market data for the market "ETH/DEC24" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start | auction end |
      | 1000       | TRADING_MODE_CONTINUOUS | 30      | 994       | 1006      | 74340        | 180000000      | 1             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 991       | 1009      | 74340        | 180000000      | 1             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 90      | 989       | 1011      | 74340        | 180000000      | 1             | 0             | 0           |
      | 1000       | TRADING_MODE_CONTINUOUS | 120     | 988       | 1012      | 74340        | 180000000      | 1             | 0             | 0           |

    # Place 2 persistent orders that are outside all of the price bounds
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC24 | buy  | 1      | 1013  | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | ETH/DEC24 | sell | 1      | 1013  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    When the network moves ahead "1" blocks

    # Check we have been placed in auction
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC24"

    And the starting auction time for market "ETH/DEC24" is "1704067221000000000"
    And the ending auction time for market "ETH/DEC24" is "1704067231000000000"

    # Check we know the auction time
    And the market data for the market "ETH/DEC24" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start       | auction end         |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 991       | 1009      | 150612       | 180000000      | 1             | 1704067221000000000 | 1704067231000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 90      | 989       | 1011      | 150612       | 180000000      | 1             | 1704067221000000000 | 1704067231000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 988       | 1012      | 150612       | 180000000      | 1             | 1704067221000000000 | 1704067231000000000 |

    # Now update the risk model to deactivate all pending price bounds
    Then the markets are updated:
      | id        | risk model                 |
      | ETH/DEC24 | my-log-normal-risk-model-2 |

    # If we move ahead 10 blocks we should come out of auction instead of it being extended
    When the network moves ahead "15" blocks


    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC24"

    And the market data for the market "ETH/DEC24" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction start | auction end |
      | 1013       | TRADING_MODE_CONTINUOUS         | 30      | 1007      | 1019      | 225372       | 180000000      | 2             | 0             | 0           |
      | 1013       | TRADING_MODE_CONTINUOUS         | 60      | 1004      | 1022      | 225372       | 180000000      | 2             | 0             | 0           |
      | 1013       | TRADING_MODE_CONTINUOUS         | 90      | 1002      | 1024      | 225372       | 180000000      | 2             | 0             | 0           |
      | 1013       | TRADING_MODE_CONTINUOUS         | 120     | 1001      | 1026      | 225372       | 180000000      | 2             | 0             | 0           |


    # The mark price should show the orders have traded
    And the mark price should be "1013" for the market "ETH/DEC24"

   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status         |
    | party1 | ETH/DEC24   | buy1      | buy  | 1      | 0         | 1013  | STATUS_FILLED  |
    | party5 | ETH/DEC24   | sell1     | sell | 1      | 0         | 1013  | STATUS_FILLED  |

