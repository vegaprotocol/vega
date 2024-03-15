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
      | 60      | 0.999       | 10                |
      | 120     | 0.9995      | 5                 |

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

  Scenario: Persistent order results in an auction (1 out of 2 triggers breached), orders placed during auction result in trade
            with indicative price outside the price monitoring bounds of the 2nd trigger, hence auction get extended (by extension
            period specified for the 2nd trigger), additional orders resulting in more trades (indicative price still outside the
            2nd trigger bounds) placed, auction concludes. (0032-PRIM-027)

    # Check that the market price bounds are set 
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 995       | 1005      | 0            | 0              | 0             |
      | 1000       | TRADING_MODE_CONTINUOUS | 120     | 992       | 1008      | 0            | 0              | 0             |

    # Place persistent orders to breach the first trigger level (995->1005)
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1006  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1006  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # We are in auction, place crossing orders outside the level of the second trigger (992->1008)
    # This should trigger an auction extension for 10 more blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1009  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1009  | 0                | TYPE_LIMIT | TIF_GTC |

    # Move forward 5 blocks should not end the auction due to the extension
    When the network moves ahead "5" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # Add some more order that will match at uncrossing time
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1007  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1007  | 0                | TYPE_LIMIT | TIF_GTC |

    # Move forward 10 blocks should end the auction
    When the network moves ahead "10" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

