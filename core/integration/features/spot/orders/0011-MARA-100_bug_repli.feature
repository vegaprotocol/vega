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
      | USD | 0              |
      | BTC | 0              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 36000   | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/USD | BTC/USD | BTC        | USD         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 0              | 0                       | default-basic |

    And the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 2     |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | validators.epoch.length                          | 58s   |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | USD   | 50000000 |
      | party1 | BTC   | 100      |
      | party2 | USD   | 1000     |
      | party3 | USD   | 1000     |
      | party2 | BTC   | 100      |
      | lp     | USD   | 1000     |
      | lp     | BTC   | 100      |
    And the average block duration is "1"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp    | BTC/USD   | 5                 | 0.025 | submission |

    Then "party2" should have general account balance of "1000" for asset "USD"
    Then "party2" should have general account balance of "100" for asset "BTC"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/USD   | buy  | 100    | 55000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/USD   | buy  | 3      | 59000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/USD   | buy  | 1      | 60000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 1      | 60000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 3      | 61000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 3      | 62000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 4      | 63000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 10     | 64000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 5      | 65000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "2" blocks

    Then the market data for the market "BTC/USD" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | open interest |
      | 60000      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 52466     | 68503     | 0             |

    Then "party2" should have general account balance of "61000" for asset "USD"
    Then "party2" should have general account balance of "74" for asset "BTC"

    Then "party2" should have holding account balance of "25" for asset "BTC"


    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference | error |
      | party3 | BTC/USD   | buy  | 20     | 0     | 4                | TYPE_MARKET | TIF_IOC |           |       |

    Then the network moves ahead "1" blocks

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party3 | 64000 | 10   | party2 |
      | party3 | 63000 | 4    | party2 |
      | party3 | 62000 | 3    | party2 |
      | party3 | 61000 | 3    | party2 |

    Then "party2" should have general account balance of "62000" for asset "USD"
    Then "party2" should have general account balance of "74" for asset "BTC"

    Then "party2" should have holding account balance of "5" for asset "BTC"

    Then "party3" should have general account balance of "0" for asset "USD"
    Then "party3" should have general account balance of "20" for asset "BTC"


