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
      | party2 | ETH   | 10000  |
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


  Scenario: For an iceberg order that is submitted with total size x and display size y the holding asset taken
            should be identical to a regular order of size x rather than one of size y (0014-ORDT-091)


    # Place a normal order of size x and see how much is placed in the holding account
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type         | tif     | reference |
      | party2 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT   | TIF_GTC | buy1      |

    Then "party2" should have general account balance of "9900" for asset "ETH"
    Then "party2" should have holding account balance of "100" for asset "ETH"

    # Cancel the order so all funds are returned to the general account
    Then the parties cancel the following orders:
      | party  | reference  |
      | party2 | buy1 |

    Then "party2" should have general account balance of "10000" for asset "ETH"
    Then "party2" should have holding account balance of "0" for asset "ETH"

    # Place an iceberg order with size x
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only |
      | party2 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | 5         | 2                    | post |

    Then "party2" should have general account balance of "9900" for asset "ETH"
    Then "party2" should have holding account balance of "100" for asset "ETH"
