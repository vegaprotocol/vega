Feature: Spot market

  Background:

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.value.windowLength               | 1h    |

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
      | party5 | BTC   | 1000   |
    And the average block duration is "1"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error                                     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |                                           |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |                                           |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

  Scenario: Order reason of ORDER_ERROR_INVALID_MARKET_ID when sending an order with an invalid market ID (0024-OSTA-046)

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                         |
      | party1 | BTC2/ETH2 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy       | OrderError: Invalid Market ID |
      | party2 | BTC2/ETH2 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell      | OrderError: Invalid Market ID |

