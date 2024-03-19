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
      | 60      | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100000 |
      | party2 | ETH   | 100000 |
      | party4 | BTC   | 10000  |
      | party5 | BTC   | 10000  |
    And the average block duration is "1"

  Scenario: When leaving an auction, all GFA orders will be cancelled. (0026-AUCT-032)

    # Place some orders that cross so we can leave the auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | buy1      |
      | party2 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GFA | buy2      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |
      | party4 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GFA | sell2     |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    # Check that all the GFA orders have been either matched or cancelled
    And the orders should have the following states:
      | party  | market id | reference | side | volume | remaining | price | status           |
      | party1 | BTC/ETH   | buy1      | buy  | 1      | 0         | 1000  | STATUS_FILLED    |
      | party2 | BTC/ETH   | buy2      | buy  | 1      | 1         | 999   | STATUS_CANCELLED |
      | party4 | BTC/ETH   | sell2     | sell | 1      | 1         | 1001  | STATUS_CANCELLED |
      | party5 | BTC/ETH   | sell1     | sell | 1      | 0         | 1000  | STATUS_FILLED    |

    # Move into a price monitoring auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1020  | 0                | TYPE_LIMIT | TIF_GTC | buy3      |
      | party5 | BTC/ETH   | sell | 1      | 1020  | 0                | TYPE_LIMIT | TIF_GTC | sell3     |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # Place some GFA orders
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GFA | buy4      |
      | party4 | BTC/ETH   | sell | 10     | 1001  | 0                | TYPE_LIMIT | TIF_GFA | sell4     |

    # Wait for us to move out of auction
    When the network moves ahead "10" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # All the GFA orders should be cancelled 
    And the orders should have the following states:
      | party  | market id | reference | side | volume | remaining | price | status           |
      | party2 | BTC/ETH   | buy4      | buy  | 1      | 1         | 999   | STATUS_CANCELLED |
      | party4 | BTC/ETH   | sell4     | sell | 10     | 9         | 1001  | STATUS_CANCELLED |

