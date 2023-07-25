Feature: Test decimal places in LP order, liquidity provider reward distribution; Should also cover liquidity-fee-setting and equity-like-share calc and total stake.

  Scenario: 001: 0070-MKTD-007, 0042-LIQF-001, 0018-RSKM-005, 0018-RSKM-008
  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 18    |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
      | USD | 2              |
    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       | 1e6                    | 1e6                       |
      | USD/DEC19 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       | 1e6                    | 1e6                       |
      | USD/DEC20 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       | 1e6                    | 1e6                       |
      | USD/DEC21 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       | 1e6                    | 1e6                       |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount          |
      | lp1    | USD   | 100000000000    |
      | lp1    | ETH   | 100000000000000 |
      | party1 | USD   | 10000000000     |
      | party1 | ETH   | 10000000000000  |
      | party2 | USD   | 10000000000     |
      | party2 | ETH   | 10000000000000  |
      | lpprov | ETH   | 100000000000000 |
      | lpprov | USD   | 100000000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1    | USD/DEC19 | 1000000000        | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000000        | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000000        | 0.001 | submission |
      | lp1 | lp1    | USD/DEC19 | 1000000000        | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000000        | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000000        | 0.001 | submission |
      | lp1 | lp1    | USD/DEC19 | 1000000000        | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000000        | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000000        | 0.001 | submission |
      | lp1 | lp1    | USD/DEC19 | 1000000000        | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000000        | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000000        | 0.001 | submission |
      | lp4 | lpprov | USD/DEC19 | 5000000000        | 0.001 | submission |
      | lp5 | lpprov | USD/DEC20 | 5000000000        | 0.001 | submission |
      | lp6 | lpprov | USD/DEC21 | 5000000000        | 0.001 | submission |
      | lp4 | lpprov | USD/DEC19 | 5000000000        | 0.001 | submission |
      | lp5 | lpprov | USD/DEC20 | 5000000000        | 0.001 | submission |
      | lp6 | lpprov | USD/DEC21 | 5000000000        | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | BID              | 1          | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 2          | 1000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | MID              | 2          | 100000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | MID              | 2          | 100000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 1          | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | sell | ASK              | 1          | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | sell | ASK              | 1          | 200000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | MID              | 2          | 1000   |
      | lp1    | USD/DEC20 | 2         | 1                    | sell | MID              | 2          | 100000 |
      | lp1    | USD/DEC21 | 2         | 1                    | sell | MID              | 2          | 100000 |
      | lpprov | USD/DEC19 | 2         | 1                    | buy  | BID              | 1          | 2000   |
      | lpprov | USD/DEC20 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lpprov | USD/DEC21 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lpprov | USD/DEC19 | 2         | 1                    | sell | MID              | 2          | 1000   |
      | lpprov | USD/DEC20 | 2         | 1                    | sell | MID              | 2          | 100000 |
      | lpprov | USD/DEC21 | 2         | 1                    | sell | MID              | 2          | 100000 |

    Then the parties place the following orders:
      | party  | market id | side | volume  | price     | resulting trades | type       | tif     |
      | party1 | USD/DEC19 | buy  | 1000    | 900000    | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC20 | buy  | 100000  | 90000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC21 | buy  | 1000    | 90000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC19 | buy  | 10000   | 1000000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC20 | buy  | 1000000 | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC21 | buy  | 10000   | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC19 | sell | 1000    | 1100000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC20 | sell | 100000  | 110000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC21 | sell | 1000    | 110000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC19 | sell | 10000   | 1000000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC20 | sell | 1000000 | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC21 | sell | 10000   | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "USD/DEC19"
    Then the opening auction period ends for market "USD/DEC20"
    Then the opening auction period ends for market "USD/DEC21"

    And the market data for the market "USD/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000000    | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3556900000   | 6000000000     | 10000         |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569 *100 = 3556900 (using asset decimal)

    And the market data for the market "USD/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000000  | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3556900000   | 6000000000     | 1000000       |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569

    And the market data for the market "USD/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000000  | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3556900000   | 6000000000     | 10000         |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569
    # max_oi: max open interest

    Then the order book should have the following volumes for market "USD/DEC19":
      | side | price   | volume |
      | buy  | 898000  | 59392  |
      | buy  | 900000  | 1000   |
      | buy  | 999000  | 6674   |
      | sell | 1001000 | 56612  |
      | sell | 1100000 | 1000   |
      | sell | 1102000 | 3025   |

    Then the order book should have the following volumes for market "USD/DEC20":
      | side | price     | volume  |
      | buy  | 89800000  | 5939125 |
      | buy  | 90000000  | 100000  |
      | buy  | 99900000  | 667335  |
      | sell | 100100000 | 5661006 |
      | sell | 110000000 | 100000  |
      | sell | 110200000 | 302481  |

    Then the order book should have the following volumes for market "USD/DEC21":
      | side | price     | volume |
      | buy  | 89800000  | 59392  |
      | buy  | 90000000  | 1000   |
      | buy  | 99900000  | 6674   |
      | sell | 100100000 | 56612  |
      | sell | 110000000 | 1000   |
      | sell | 110200000 | 3025   |

    And the liquidity provider fee shares for the market "USD/DEC19" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.1666666666666667 | 1000000000              |

    And the liquidity provider fee shares for the market "USD/DEC20" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.1666666666666667 | 1000000000              |

    And the liquidity provider fee shares for the market "USD/DEC21" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.1666666666666667 | 1000000000              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin     | general        | bond       |
      | lp1    | ETH   | USD/DEC19 | 4134260182 | 99984597723110 | 1000000000 |
      | lp1    | USD   |           |            | 100000000000   |            |
      | lp1    | ETH   | USD/DEC20 | 4133756526 | 99984597723110 | 1000000000 |
      | lp1    | USD   |           |            | 100000000000   |            |
      | lp1    | ETH   | USD/DEC21 | 4134260182 | 99984597723110 | 1000000000 |
      | lp1    | USD   |           |            | 100000000000   |            |
      | party1 | ETH   | USD/DEC19 | 1047352496 | 9996857942512  |            |
      | party1 | USD   |           |            | 10000000000    |            |
      | party2 | ETH   | USD/DEC19 | 4737795584 | 9985786613248  |            |
      | party2 | USD   |           |            | 10000000000    |            |

  Scenario: 002: 0070-MKTD-007, 0042-LIQF-001, 0038-OLIQ-002; 0038-OLIQ-006; 0019-MCAL-008, check updated version of dpd feature in 0038-OLIQ-liquidity_provision_order_type.md

  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
      | USD | 2              |
    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor: 4.556903591579
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       | 1e6                    | 1e6                       |
      | USD/DEC19 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       | 1e6                    | 1e6                       |
      | USD/DEC20 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       | 1e6                    | 1e6                       |
      | USD/DEC21 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       | 1e6                    | 1e6                       |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount          |
      | lp1    | USD   | 100000000000    |
      | lp1    | ETH   | 100000000000000 |
      | party1 | USD   | 10000000000     |
      | party1 | ETH   | 10000000000000  |
      | party2 | USD   | 10000000000     |
      | party2 | ETH   | 10000000000000  |
      | lpprov | ETH   | 100000000000000 |
      | lpprov | USD   | 100000000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1    | USD/DEC19 | 1000000           | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000           | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000           | 0.001 | submission |
      | lp1 | lp1    | USD/DEC19 | 1000000           | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000           | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000           | 0.001 | submission |
      | lp1 | lp1    | USD/DEC19 | 1000000           | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000           | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000           | 0.001 | submission |
      | lp1 | lp1    | USD/DEC19 | 1000000           | 0.001 | submission |
      | lp2 | lp1    | USD/DEC20 | 1000000           | 0.001 | submission |
      | lp3 | lp1    | USD/DEC21 | 1000000           | 0.001 | submission |
      | lp4 | lpprov | USD/DEC19 | 5000000000        | 0.001 | submission |
      | lp5 | lpprov | USD/DEC20 | 5000000000        | 0.001 | submission |
      | lp6 | lpprov | USD/DEC21 | 5000000000        | 0.001 | submission |
      | lp4 | lpprov | USD/DEC19 | 5000000000        | 0.001 | submission |
      | lp5 | lpprov | USD/DEC20 | 5000000000        | 0.001 | submission |
      | lp6 | lpprov | USD/DEC21 | 5000000000        | 0.001 | submission |
    And the parties place the following pegged iceberg orders: 
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | BID              | 1          | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 2          | 1000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | MID              | 2          | 100000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | MID              | 2          | 100000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 1          | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | sell | ASK              | 1          | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | sell | ASK              | 1          | 200000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | MID              | 2          | 1000   |
      | lp1    | USD/DEC20 | 2         | 1                    | sell | MID              | 2          | 100000 |
      | lp1    | USD/DEC21 | 2         | 1                    | sell | MID              | 2          | 100000 |
      | lpprov | USD/DEC19 | 2         | 1                    | buy  | BID              | 1          | 2000   |
      | lpprov | USD/DEC20 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lpprov | USD/DEC21 | 2         | 1                    | buy  | BID              | 1          | 200000 |
      | lpprov | USD/DEC19 | 2         | 1                    | sell | MID              | 2          | 1000   |
      | lpprov | USD/DEC20 | 2         | 1                    | sell | MID              | 2          | 100000 |
      | lpprov | USD/DEC21 | 2         | 1                    | sell | MID              | 2          | 100000 |

    Then the parties place the following orders:
      | party  | market id | side | volume  | price     | resulting trades | type       | tif     |
      | party1 | USD/DEC19 | buy  | 1000    | 900000    | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC20 | buy  | 100000  | 90000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC21 | buy  | 1000    | 90000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC19 | buy  | 10000   | 1000000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC20 | buy  | 1000000 | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC21 | buy  | 10000   | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC19 | sell | 1000    | 1100000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC20 | sell | 100000  | 110000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC21 | sell | 1000    | 110000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC19 | sell | 10000   | 1000000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC20 | sell | 1000000 | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC21 | sell | 10000   | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "USD/DEC19"
    Then the opening auction period ends for market "USD/DEC20"
    Then the opening auction period ends for market "USD/DEC21"

    And the market data for the market "USD/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000000    | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3556900000   | 5001000000     | 10000         |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569 *100000 = 3556900000 (using asset decimal)

    And the market data for the market "USD/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000000  | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3556900000   | 5001000000     | 1000000       |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569 *100000 = 3556900000 (using asset decimal)

    And the market data for the market "USD/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000000  | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3556900000   | 5001000000     | 10000         |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569 *100000 = 3556900000 (using asset decimal)
    # max_oi: max open interest

    # could be improved: as we do have fractional order, if we do the position scaling before we divide by price we can get a more sensible result
    Then the order book should have the following volumes for market "USD/DEC19":
      | side | price   | volume |
      | buy  | 898000  | 55684  |
      | buy  | 900000  | 1000   |
      | buy  | 999000  | 7      |
      | sell | 1001000 | 49958  |
      | sell | 1100000 | 1000   |
      | sell | 1102000 | 4      |

    Then the order book should have the following volumes for market "USD/DEC20":
      | side | price     | volume  |
      | buy  | 89800000  | 5568301 |
      | buy  | 90000000  | 100000  |
      | buy  | 99900000  | 668     |
      | sell | 100100000 | 4995672 |
      | sell | 110000000 | 100000  |
      | sell | 110200000 | 303     |

    Then the order book should have the following volumes for market "USD/DEC21":
      | side | price     | volume |
      | buy  | 89800000  | 55684  |
      | buy  | 90000000  | 1000   |
      | buy  | 99900000  | 7      |
      | sell | 100100000 | 49958  |
      | sell | 110000000 | 1000   |
      | sell | 110200000 | 4      |

    And the liquidity provider fee shares for the market "USD/DEC19" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0001999600079984 | 1000000                 |

    And the liquidity provider fee shares for the market "USD/DEC20" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0001999600079984 | 1000000                 |

    And the liquidity provider fee shares for the market "USD/DEC21" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0001999600079984 | 1000000                 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin     | general        | bond    |
      | lp1    | ETH   | USD/DEC19 | 4695112    | 99999983469540 | 1000000 |
      | lp1    | USD   |           |            | 100000000000   |         |
      | lp1    | ETH   | USD/DEC20 | 4140236    | 99999983469540 | 1000000 |
      | lp1    | USD   |           |            | 100000000000   |         |
      | lp1    | ETH   | USD/DEC21 | 4695112    | 99999983469540 | 1000000 |
      | lp1    | USD   |           |            | 100000000000   |         |
      | party1 | ETH   | USD/DEC19 | 1179036394 | 9996462886930  |         |
      | party1 | USD   |           |            | 10000000000    |         |
      | party2 | ETH   | USD/DEC19 | 4737795584 | 9985786613248  |         |
      | party2 | USD   |           |            | 10000000000    |         |

    Then the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     |
      | party1 | USD/DEC19 | buy  | 5      | 1001000 | 1                | TYPE_LIMIT | TIF_GTC |

    And the accumulated liquidity fees should be "501" for the market "USD/DEC19"
    # liquidity fee = 5 * 1001000 * 0.001 which means actual number without decimal is 0.005*1001*0.001 = 0.005005, and translate back into asset decimal 501, given asset decimal 5, market decimal 3, position decimal 3

    Then the parties place the following orders:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     |
      | party1 | USD/DEC20 | buy  | 5      | 100100000 | 1                | TYPE_LIMIT | TIF_GTC |
    And the accumulated liquidity fees should be "6" for the market "USD/DEC20"
    # liquidity fee = 5 * 100100000 * 0.001 which means actual number without decimal is 0.00005*1001*0.001 = 0.00005005, and translate back into asset decimal 5.005 (given fee is rounded up in vega, so it should be 6) given asset decimal 5, market decimal 5, position decimal 5

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     |
      | party1 | USD/DEC21 | buy  | 1      | 100100000 | 1                | TYPE_LIMIT | TIF_GTC |
    And the accumulated liquidity fees should be "101" for the market "USD/DEC21"
    # liquidity fee = 1 * 100100000 * 0.001 which means actual number without decimal is 0.001*1001*0.001 = 0.001001, and translate back into asset decimal 100.1 (given fee is rounded up in vega, so it should be 101) given asset decimal 5, market decimal 5, position decimal 3

    #check MTM settlement with correct PDP
    And the market data for the market "USD/DEC21" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100100000  | 100100000         | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3560812945   | 5001000000     | 10001         |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1001 x 10.001 x 1 x 3.5569=35608.1294569, which is 3560812945 in asset decimal (which is 5)

    And the parties should have the following account balances:
      | party | asset | market id | margin  | general        | bond    |
      | lp1   | ETH   | USD/DEC21 | 5122062 | 99999980907847 | 1000000 |
      | lp1   | USD   |           |         | 100000000000   |         |

    # amend LP commintment amount
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | USD/DEC19 | 2000000           | 0.001 | amendment |
      | lp1 | lp1   | USD/DEC19 | 2000000           | 0.001 | amendment |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/DEC19 | 2         | 1                    | buy  | MID              | 2          | 100000 |
      | lp1   | ETH/DEC19 | 2         | 1                    | sell | ASK              | 1          | 200000 |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 5002000000     | 10005         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1    | USD/DEC19 | 4000000000        | 0.001 | amendment |
      | lp1 | lp1    | USD/DEC19 | 4000000000        | 0.001 | amendment |
      | lp4 | lpprov | USD/DEC19 | 1000000000        | 0.001 | amendment |
      | lp4 | lpprov | USD/DEC19 | 1000000000        | 0.001 | amendment |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 2          | 100000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 1          | 200000 |
      | lpprov | USD/DEC19 | 2         | 1                    | buy  | BID              | 1          | 2000   |
      | lpprov | USD/DEC19 | 2         | 1                    | sell | MID              | 2          | 1000   |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 5000000000     | 10005         |

    #reduce LP commitment amount
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | USD/DEC19 | 3600000000        | 0.001 | amendment |
      | lp1 | lp1   | USD/DEC19 | 3600000000        | 0.001 | amendment |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | USD/DEC19 | 2         | 1                    | buy  | MID              | 2          | 100000 |
      | lp1   | USD/DEC19 | 2         | 1                    | sell | ASK              | 1          | 200000 |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 4600000000     | 10005         |

    # 0038-OLIQ-006 assure that submission bringing supplied stake < target stake gets rejected
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   | reference        | error                                            |
      | lp1 | lp1   | USD/DEC19 | 2562237127        | 0.001 | amendment | failing_amedment | commitment submission rejected, not enough stake |
      | lp1 | lp1   | USD/DEC19 | 2562237127        | 0.001 | amendment |                  |                                                  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type      | reference            | error                                            |
      | lp1 | lp1   | USD/DEC19 | 2562237127        | 0.001 | cancellation | failing_cancellation | commitment submission rejected, not enough stake |
      | lp1 | lp1   | USD/DEC19 | 2562237127        | 0.001 | cancellation |                      |                                                  |

  Scenario: 003, no decimal, 0042-LIQF-001

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |
    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | USD        | USD   | log-normal-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |

    Given the average block duration is "2"

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |
      | lpprov | USD   | 1000000000 |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 40000             | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 40000             | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 40000             | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 40000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1          | 2      |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 2          | 1      |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1          | 2      |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 2          | 1      |
 
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100000  | 864       | 1154      | 35569        | 40000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569
    # max_oi: max open interest

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 15     |
      | buy  | 900   | 1      |
      | buy  | 999   | 27     |
      | sell | 1001  | 27     |
      | sell | 1100  | 1      |
      | sell | 1102  | 13     |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 40000                   |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 170732 | 999789268 | 40000 |
      | party1 | USD   | ETH/MAR22 | 10473  | 99989527  |       |
      | party2 | USD   | ETH/MAR22 | 47378  | 99952622  |       |

    Then the network moves ahead "1" blocks

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 864       | 1154      |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
