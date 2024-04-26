Feature: Spot market bug (from the incentive) replication

  Scenario: 0029-FEES-015: Fees are collected during continuous trading and auction modes and distributed to the appropriate accounts, as described above.

  Background:

    Given the following network parameters are set:
      | name                                            | value |
      | network.markPriceUpdateMaximumFrequency         | 1s    |
      | market.value.windowLength                       | 1h    |
      | market.liquidity.maximumLiquidityFeeFactorLevel | 0.4   |
      | validators.epoch.length                         | 4s    |

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
      | 36000   | 0.999       | 1                 |
      | 38000   | 0.999       | 2                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    And the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 2     |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | validators.epoch.length                          | 58s   |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100000 |
      | party1 | BTC   | 10000  |
      | party2 | ETH   | 100000 |
      | party3 | ETH   | 100000 |
      | party2 | BTC   | 10000  |
      | lp     | ETH   | 100000 |
      | lp     | BTC   | 10000  |
    And the average block duration is "1"

    # No orders have been places so we shouldn't have any holding accounts
    And "party1" should have only the following accounts:
      | type                 | asset | amount |
      | ACCOUNT_TYPE_GENERAL | ETH   | 100000 |
      | ACCOUNT_TYPE_GENERAL | BTC   | 10000  |

    And "party2" should have only the following accounts:
      | type                 | asset | amount |
      | ACCOUNT_TYPE_GENERAL | ETH   | 100000 |
      | ACCOUNT_TYPE_GENERAL | BTC   | 10000  |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp    | BTC/ETH   | 500               | 0.025 | submission |
    # Place some orders to create the holding accounts
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 10000  | 1500  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1000   | 1600  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 400    | 1700  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 300    | 1800  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 300    | 1900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 100    | 2100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 100    | 2200  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 100    | 2300  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "2" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | open interest |
      | 2000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 1749      | 2283      | 0             |
      | 2000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 38000   | 1743      | 2291      | 0             |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | party3 | BTC/ETH   | buy  | 2000   | 0     | 1                | TYPE_MARKET | TIF_IOC |           |

    Then the network moves ahead "1" blocks

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party3 | 2100  | 20   | party2 |
