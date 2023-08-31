Feature: Test LP bond account when market is terminated

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE19" named "ethDec19Oracle":
      | property         | type         | binding         | decimals |
      | prices.ETH.value | TYPE_INTEGER | settlement data | 0        |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE19" named "ethDec19Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the following assets are registered:
      | id  | decimal places |
      | USD | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 360000  | 0.99        | 3                 |
    And the following network parameters are set:
      | name                                                  | value |
      | market.value.windowLength                             | 60s   |
      | market.stake.target.timeWindow                        | 20s   |
      | market.stake.target.scalingFactor                     | 1     |
      | market.liquidity.targetstake.triggering.ratio         | 0.5   |
      | network.markPriceUpdateMaximumFrequency               | 0s    |
      | limits.markets.maxPeggedOrders                        | 6     |
      | market.auction.minimumDuration                        | 1     |
      | market.fee.factors.infrastructureFee                  | 0.001 |
      | market.fee.factors.makerFee                           | 0.004 |
      | market.liquidityV2.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                               | 5s    |
      | market.liquidityV2.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength          | 1h    |
      | market.liquidityV2.sla.nonPerformanceBondPenaltySlope | 0.19  |
      | market.liquidityV2.sla.nonPerformanceBondPenaltyMax   | 1     |
      | validators.epoch.length                               | 2s    |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | providers fee calculation time step | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 10                                  | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | ethDec19Oracle     | 1e1                    | 1e0                       | SLA        |

    Given the average block duration is "2"
  @Now @SLABond
  Scenario: 001: All liquidity providers in the market receive a greater than zero amount of liquidity fee
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 159000 |
      | lp2    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 3000              | 0.002 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 1000              | 0.001 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 5         | 1                    | buy  | BID              | 5      | 20     |
      | lp1   | ETH/MAR22 | 5         | 1                    | sell | ASK              | 5      | 20     |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 2      | 20     |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 2      | 20     |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 360000  | 756       | 1309      | 3556         | 4000           | 1             |
    # # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569036*10000

    And the liquidity fee factor should be "0.002" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.75              | 3000                    |
      | lp2   | 0.25              | 4000                    |

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 26677  | 129323  | 3000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 88329   | 1000 |
    #intial margin lp2: 2*1000*3.5569036*1.5=10670

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | lp1    | 0      | 0              | 0            |
      | lp2    | 0      | 0              | 0            |
      | party1 | 1      | 0              | 0            |
      | party2 | -1     | 0              | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 15     | 1120  | 2                | TYPE_LIMIT | TIF_GTC | buy-1     |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 360000  | 756       | 1309      | 56910        | 4000           | 16            |

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 26677  | 129346  | 3000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 88329   | 1000 |

    When the network moves ahead "2" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | lp1    | -5     | 0              | 0            |
      | lp2    | 0      | 0              | 0            |
      | party1 | 16     | 320            | 0            |
      | party2 | -11    | -320           | 0            |

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 155878 | 145     | 2430 |
      | lp2   | USD   | ETH/MAR22 | 0      | 99000   | 1000 |

    When the oracles broadcast data signed with "0xCAFECAFE19":
      | name               | value |
      | trading.terminated | true  |
    #And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1120       | TRADING_MODE_NO_TRADING | 360000  | 831       | 1440      | 63739        | 3430           | 16            |

    Then the oracles broadcast data signed with "0xCAFECAFE19":
      | name             | value |
      | prices.ETH.value | 1600 |
    And the network moves ahead "3" blocks

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | lp1    | -5     | 0              | -2400        |
      | lp2    | 0      | 0              | 0            |
      | party1 | 16     | 0              | 8000         |
      | party2 | -11    | 0              | -5600        |

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 0      | 155592  | 0    |
      | lp2   | USD   | ETH/MAR22 | 0      | 99810   | 0    |

    And the insurance pool balance should be "1255" for the market "ETH/MAR22"

