Feature: Test decimal places in LP order, liquidity provider reward distribution; Should also cover liquidity-fee-setting and equity-like-share calc and total stake.

  @SLABug
  Scenario: 001: 0070-MKTD-007, 0042-LIQF-001, 0018-RSKM-005, 0018-RSKM-008
    Given the following network parameters are set:
      | name                                             | value |
      | market.value.windowLength                        | 1h    |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | limits.markets.maxPeggedOrders                   | 18    |
      | market.liquidity.providersFeeCalculationTimeStep | 600s  |
      | market.liquidity.equityLikeShareFeeFraction      | 1     |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.0              | 24h         | 1.0            |
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
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       | 0.5                    | 0                         | SLA        |
      | USD/DEC19 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       | 0.5                    | 0                         | SLA        |
      | USD/DEC20 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       | 0.5                    | 0                         | SLA        |
      | USD/DEC21 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       | 0.5                    | 0                         | SLA        |

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
      | lp4 | lpprov | USD/DEC19 | 5000000000        | 0.001 | submission |
      | lp5 | lpprov | USD/DEC20 | 5000000000        | 0.001 | submission |
      | lp6 | lpprov | USD/DEC21 | 5000000000        | 0.001 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | BID              | 1      | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 2      | 1000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 1      | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | sell | ASK              | 1      | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | sell | ASK              | 1      | 200000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | MID              | 2      | 1000   |
      | lp1    | USD/DEC20 | 2         | 1                    | sell | MID              | 2      | 100000 |
      | lp1    | USD/DEC21 | 2         | 1                    | sell | MID              | 2      | 100000 |
      | lpprov | USD/DEC19 | 2         | 1                    | buy  | BID              | 1      | 2000   |
      | lpprov | USD/DEC20 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lpprov | USD/DEC21 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lpprov | USD/DEC19 | 2         | 1                    | sell | MID              | 2      | 1000   |
      | lpprov | USD/DEC20 | 2         | 1                    | sell | MID              | 2      | 100000 |
      | lpprov | USD/DEC21 | 2         | 1                    | sell | MID              | 2      | 100000 |

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

    When the network moves ahead "2" blocks

    Then the market data for the market "USD/DEC19" should be:
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
      | buy  | 898000  | 2      |
      | buy  | 900000  | 1000   |
      | buy  | 999000  | 2      |
      | sell | 1001000 | 4      |
      | sell | 1100000 | 1000   |
      | sell | 1102000 | 1      |

    Then the order book should have the following volumes for market "USD/DEC20":
      | side | price     | volume |
      | buy  | 89800000  | 2      |
      | buy  | 90000000  | 100000 |
      | buy  | 99900000  | 2      |
      | sell | 100100000 | 4      |
      | sell | 110000000 | 100000 |
      | sell | 110200000 | 1      |

    Then the order book should have the following volumes for market "USD/DEC21":
      | side | price     | volume |
      | buy  | 89800000  | 2      |
      | buy  | 90000000  | 1000   |
      | buy  | 99900000  | 2      |
      | sell | 100100000 | 4      |
      | sell | 110000000 | 1000   |
      | sell | 110200000 | 1      |

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
      | lp1    | ETH   | USD/DEC19 | 1280486    | 99996997426223 | 1000000000 |
      | lp1    | USD   |           |            | 100000000000   |            |
      | lp1    | ETH   | USD/DEC20 | 12805      | 99996997426223 | 1000000000 |
      | lp1    | USD   |           |            | 100000000000   |            |
      | lp1    | ETH   | USD/DEC21 | 1280486    | 99996997426223 | 1000000000 |
      | lp1    | USD   |           |            | 100000000000   |            |
      | party1 | ETH   | USD/DEC19 | 1656961234 | 9995029116298  |            |
      | party1 | USD   |           |            | 10000000000    |            |
      | party2 | ETH   | USD/DEC19 | 5295112741 | 9984114661777  |            |
      | party2 | USD   |           |            | 10000000000    |            |

  @SLABug
  Scenario: 002: 0070-MKTD-007, 0042-LIQF-001, 0019-MCAL-008, check updated version of dpd feature in 0038-OLIQ-liquidity_provision_order_type.md
    Given the following network parameters are set:
      | name                                          | value |
      | market.value.windowLength                     | 1h    |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | limits.markets.maxPeggedOrders                | 30    |
      | market.liquidity.earlyExitPenalty           | 1.0   |
      | market.liquidity.bondPenaltyParameter       | 1.0   |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 1.0              | 24h         | 1.0            |
    

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
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       | 0.5                    | 0                         | SLA        |
      | USD/DEC19 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       | 0.5                    | 0                         | SLA        |
      | USD/DEC20 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       | 0.5                    | 0                         | SLA        |
      | USD/DEC21 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       | 0.5                    | 0                         | SLA        |

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
      | lp4 | lpprov | USD/DEC19 | 5000000000        | 0.001 | submission |
      | lp5 | lpprov | USD/DEC20 | 5000000000        | 0.001 | submission |
      | lp6 | lpprov | USD/DEC21 | 5000000000        | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | BID              | 1      | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 2      | 1000   |
      | lp1    | USD/DEC20 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1    | USD/DEC21 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 1      | 2000   |
      | lp1    | USD/DEC20 | 2         | 1                    | sell | ASK              | 1      | 200000 |
      | lp1    | USD/DEC21 | 2         | 1                    | sell | ASK              | 1      | 200000 |
      | lp1    | USD/DEC19 | 8         | 1                    | sell | MID              | 8      | 1000   |
      | lp1    | USD/DEC20 | 8         | 1                    | sell | MID              | 8      | 100000 |
      | lp1    | USD/DEC21 | 8         | 1                    | sell | MID              | 8      | 100000 |
      | lpprov | USD/DEC19 | 2         | 1                    | buy  | BID              | 1      | 2000   |
      | lpprov | USD/DEC20 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lpprov | USD/DEC21 | 2         | 1                    | buy  | BID              | 1      | 200000 |
      | lpprov | USD/DEC19 | 2         | 1                    | sell | MID              | 2      | 1000   |
      | lpprov | USD/DEC20 | 2         | 1                    | sell | MID              | 2      | 100000 |
      | lpprov | USD/DEC21 | 2         | 1                    | sell | MID              | 2      | 100000 |

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

    When the network moves ahead "2" blocks

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
      | buy  | 898000  | 2      |
      | buy  | 900000  | 1000   |
      | buy  | 999000  | 2      |
      | sell | 1001000 | 10     |
      | sell | 1100000 | 1000   |
      | sell | 1102000 | 1      |

    Then the order book should have the following volumes for market "USD/DEC20":
      | side | price     | volume |
      | buy  | 89800000  | 2      |
      | buy  | 90000000  | 100000 |
      | buy  | 99900000  | 2      |
      | sell | 100100000 | 10     |
      | sell | 110000000 | 100000 |
      | sell | 110200000 | 1      |

    Then the order book should have the following volumes for market "USD/DEC21":
      | side | price     | volume |
      | buy  | 89800000  | 2      |
      | buy  | 90000000  | 1000   |
      | buy  | 99900000  | 2      |
      | sell | 100100000 | 10     |
      | sell | 110000000 | 1000   |
      | sell | 110200000 | 1      |

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
      | lp1    | ETH   | USD/DEC19 | 3841456    | 99999989278673 | 1000000 |
      | lp1    | USD   |           |            | 100000000000   |         |
      | lp1    | ETH   | USD/DEC20 | 38415      | 99999989278673 | 1000000 |
      | lp1    | USD   |           |            | 100000000000   |         |
      | lp1    | ETH   | USD/DEC21 | 3841456    | 99999989278673 | 1000000 |
      | lp1    | USD   |           |            | 100000000000   |         |
      | party1 | ETH   | USD/DEC19 | 1656961234 | 9995029116298  |         |
      | party1 | USD   |           |            | 10000000000    |         |
      | party2 | ETH   | USD/DEC19 | 5295112741 | 9984114661777  |         |
      | party2 | USD   |           |            | 10000000000    |         |


    And the accumulated liquidity fees should be "0" for the market "USD/DEC19"

    Then the parties place the following orders:
      | party  | market id | side | volume | price   | type       | tif     |
      | party1 | USD/DEC19 | buy  | 5      | 1001000 | TYPE_LIMIT | TIF_GTC |

    And the accumulated liquidity fees should be "501" for the market "USD/DEC19"
    # liquidity fee = 5 * 1001000 * 0.001 which means actual number without decimal is 0.005*1001*0.001 = 0.005005, and translate back into asset decimal 501, given asset decimal 5, market decimal 3, position decimal 3

    Then the parties place the following orders:
      | party  | market id | side | volume | price     | type       | tif     |
      | party1 | USD/DEC20 | buy  | 5      | 100100000 | TYPE_LIMIT | TIF_GTC |
    And the accumulated liquidity fees should be "6" for the market "USD/DEC20"
    # liquidity fee = 5 * 100100000 * 0.001 which means actual number without decimal is 0.00005*1001*0.001 = 0.00005005, and translate back into asset decimal 5.005 (given fee is rounded up in vega, so it should be 6) given asset decimal 5, market decimal 5, position decimal 5

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price     | type       | tif     |
      | party1 | USD/DEC21 | buy  | 1      | 100100000 | TYPE_LIMIT | TIF_GTC |
    And the accumulated liquidity fees should be "101" for the market "USD/DEC21"
    # liquidity fee = 1 * 100100000 * 0.001 which means actual number without decimal is 0.001*1001*0.001 = 0.001001, and translate back into asset decimal 100.1 (given fee is rounded up in vega, so it should be 101) given asset decimal 5, market decimal 5, position decimal 3


    And the network moves ahead "10" blocks
    #check MTM settlement with correct PDP
    And the market data for the market "USD/DEC21" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100100000  | 100100000         | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3560812945   | 5001000000     | 10001         |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1001 x 10.001 x 1 x 3.5569=35608.1294569, which is 3560812945 in asset decimal (which is 5)

    And the parties should have the following account balances:
      | party | asset | market id | margin  | general        | bond    |
      | lp1   | ETH   | USD/DEC21 | 3841456 | 99999989278976 | 1000000 |
      | lp1   | USD   |           |         | 100000000000   |         |

    # amend LP commintment amount
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | USD/DEC19 | 2000000           | 0.001 | amendment |

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | USD/DEC19 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1   | USD/DEC19 | 2         | 1                    | sell | ASK              | 1      | 200000 |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 5002000000     | 10005         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1    | USD/DEC19 | 4000000000        | 0.001 | amendment |
      | lp4 | lpprov | USD/DEC19 | 1000000000        | 0.001 | amendment |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 1      | 200000 |
      | lpprov | USD/DEC19 | 2         | 1                    | buy  | BID              | 1      | 2000   |
      | lpprov | USD/DEC19 | 2         | 1                    | sell | MID              | 2      | 1000   |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 9000000000     | 10005         |

    And the network moves ahead "10" blocks

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 1      | 200000 |
      | lpprov | USD/DEC19 | 2         | 1                    | buy  | BID              | 1      | 2000   |
      | lpprov | USD/DEC19 | 2         | 1                    | sell | MID              | 2      | 1000   |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 5000000000     | 10005         |

    #reduce LP commitment amount
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | USD/DEC19 | 3600000000        | 0.001 | amendment |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 5000000000     | 10005         |

    And the network moves ahead "10" blocks

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | USD/DEC19 | 2         | 1                    | buy  | MID              | 2      | 100000 |
      | lp1   | USD/DEC19 | 2         | 1                    | sell | ASK              | 1      | 200000 |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 4600000000     | 10005         |

    And the parties should have the following account balances:
      | party | asset | market id | margin  | general        | bond       |
      | lp1   | ETH   | USD/DEC19 | 5554318 | 99996388566114 | 3600000000 |

    # Reduce LP stake below target, results in slashing
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | USD/DEC19 | 10                | 0.001 | amendment |

    Then the network moves ahead "10" blocks

    # This should be slashed, as amendment brought market below target stake, but is currently not
    And the parties should have the following account balances:
      | party | asset | market id | margin  | general        | bond |
      | lp1   | ETH   | USD/DEC19 | 5554318 | 99997426328986 | 10   |

    And the market data for the market "USD/DEC19" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001000    | 1001000           | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3562237128   | 1000000010     | 10005         |


  Scenario: 003, no decimal, 0042-LIQF-001
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
      | name                                          | value |
      | market.value.windowLength                     | 1h    |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | limits.markets.maxPeggedOrders                | 18    |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.0              | 24h         | 1.0            |
    
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5                    | 0                         | SLA        |

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
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 2      | 1      |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 2      |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 2      | 1      |

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
      | buy  | 898   | 1      |
      | buy  | 900   | 1      |
      | buy  | 999   | 2      |
      | sell | 1001  | 2      |
      | sell | 1100  | 1      |
      | sell | 1102  | 1      |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 40000                   |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 12805  | 999947195 | 40000 |
      | party1 | USD   | ETH/MAR22 | 16570  | 99983430  |       |
      | party2 | USD   | ETH/MAR22 | 52951  | 99947049  |       |

    Then the network moves ahead "1" blocks

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 864       | 1154      |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
