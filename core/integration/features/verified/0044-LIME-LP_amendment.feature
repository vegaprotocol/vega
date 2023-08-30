Feature: Test LP mechanics when there are multiple liquidity providers;

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
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
      | 3600    | 0.99        | 3                 |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | providers fee calculation time step | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.5                          | 10                                  | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA        |

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
      | market.liquidityV2.sla.nonPerformanceBondPenaltySlope | 0.5   |
      | market.liquidityV2.sla.nonPerformanceBondPenaltyMax   | 1     |
      | validators.epoch.length                               | 10s   |

    Given the average block duration is "2"
  @Now
  Scenario: 001: lp1 and lp2 under supplies liquidity (and expects to get penalty for not meeting the SLA) since both have orders outside price range
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 50000             | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 10000             | 0.01 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 12        | 1                    | buy  | BID              | 12     | 20     | lp-b-1    |
      | lp1   | ETH/MAR22 | 12        | 1                    | sell | ASK              | 12     | 20     | lp-s-1    |
      | lp2   | ETH/MAR22 | 6         | 1                    | buy  | BID              | 6      | 20     | lp-b-2    |
      | lp2   | ETH/MAR22 | 6         | 1                    | sell | ASK              | 6      | 20     | lp-s-2    |

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
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 35569        | 45976          | 1             |
    # # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569036

    And the liquidity fee factor should be "0.02" for the market "ETH/MAR22"

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | lp1   | ETH/MAR22 | 42683       | 51219  | 64024   | 72561   |
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 64024  | 0       | 35976 |
      | lp2   | USD   | ETH/MAR22 | 32013  | 57987   | 10000 |
    #     #margin_intial lp1: 12*1000*3.5569036*1.5=64024
    Then the network moves ahead "6" blocks
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 64024  | 0       | 17988 |
      | lp2   | USD   | ETH/MAR22 | 32013  | 57987   | 5000  |

  Scenario: 002: lp1 and lp2 over supplies liquidity
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 50000             | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 10000             | 0.01 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 120       | 1                    | buy  | BID              | 120    | 20     | lp-b-1    |
      | lp1   | ETH/MAR22 | 120       | 1                    | sell | ASK              | 120    | 20     | lp-s-1    |
      | lp2   | ETH/MAR22 | 60        | 1                    | buy  | BID              | 60     | 20     | lp-b-2    |
      | lp2   | ETH/MAR22 | 60        | 1                    | sell | ASK              | 60     | 20     | lp-s-2    |
    Then the network moves ahead "2" blocks
    And the orders should have the following status:
      | party | reference | status        |
      | lp1   | lp-b-1    | STATUS_PARKED |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "2" blocks

    Then the opening auction period ends for market "ETH/MAR22"

    And the orders should have the following status:
      | party | reference | status           |
      | lp1   | lp-b-1    | STATUS_CANCELLED |
      | lp1   | lp-b-1    | STATUS_CANCELLED |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

# And the market data for the market "ETH/MAR22" should be:
#   | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
#   | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 35569        | 60000          | 1             |
# # # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569036

# And the liquidity fee factor should be "0.02" for the market "ETH/MAR22"
# `
# # And the parties should have the following margin levels:
# #   | party | market id | maintenance | search | initial | release |
# #   | lp1   | ETH/MAR22 | 42683       | 51219  | 64024   | 72561   |
# And the parties should have the following account balances:
#   | party | asset | market id | margin | general | bond  |
#   | lp1   | USD   | ETH/MAR22 | 0      | 50000   | 50000 |
#   | lp2   | USD   | ETH/MAR22 | 0      | 90000   | 10000 |
# #     #margin_intial lp1: 12*1000*3.5569036*1.5=64024
# Then the network moves ahead "6" blocks
# And the parties should have the following account balances:
#   | party | asset | market id | margin | general | bond  |
#   | lp1   | USD   | ETH/MAR22 | 0      | 50000   | 25000 |
#   | lp2   | USD   | ETH/MAR22 | 0      | 90000   | 5000  |

# And the market data for the market "ETH/MAR22" should be:
#   | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
#   | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 35569        | 30000          | 1             |
# Then debug detailed orderbook volumes for market "ETH/MAR22"
