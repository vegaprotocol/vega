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
      | party4 | BTC   | 100  |
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
    And the mark price should be "1000" for the market "BTC/ETH"

  Scenario: Amending an iceberg order to decrease size will decrease the total and remaining quantities and time priority
            of the order is not lost (0014-ORDT-102)

    # Set up a price level to have 3 orders with our iceberg in the middle
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy1      |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only | reference |
      | party1 | BTC/ETH   | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | 2         | 2                    | post | iceberg1  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |

    # Amend the iceberg to have a smaller size
    When the parties amend the following orders:
      | party  | reference  | price | size delta | tif     | 
      | party1 | iceberg1   | 1000  | -3         | TIF_GTC | 

    # Now send in a sell order, if the time order is untouched it will match against the first normal order and the iceberg
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 2      | 1000  | 2                | TYPE_LIMIT | TIF_GTC | sell1     |

    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party2 | BTC/ETH   | buy  | 1000  | 0         | 1      | buy1      | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 1000  | 1         | 2      | iceberg1  | STATUS_ACTIVE  | 
      | party2 | BTC/ETH   | buy  | 1000  | 1         | 1      | buy2      | STATUS_ACTIVE  | 
      | party5 | BTC/ETH   | sell | 1000  | 0         | 2      | sell1     | STATUS_FILLED  | 
