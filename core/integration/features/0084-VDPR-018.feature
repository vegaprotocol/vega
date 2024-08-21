Feature: A volume_discount_factors tier with differing factors across the three options has each factor set correctly (0084-VDPR-018)

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |

    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 20s         | 1.0            |

    And the following network parameters are set:
      | name                                    | value |
      | market.value.windowLength               | 60s   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 6     |
      | market.auction.minimumDuration          | 1     |


    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the volume discount program tiers named "VDP-01":
      | volume | infra factor | liquidity factor | maker factor |
      | 1000   | 0.001        | 0.002            | 0.003        |
      | 2000   | 0.005        | 0.006            | 0.007        |
      | 3000   | 0.010        | 0.012            | 0.014        |
    And the volume discount program:
      | id  | tiers  | closing timestamp | window length |
      | id1 | VDP-01 | 0                 | 4             |

    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_CONSTANT      | 0                      |

    And the fees configuration named "fees-config-2":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant | buy back fee | treasury fee |
      | 0         | 0                  | METHOD_CONSTANT      | 0.1                    | 0.001        | 0.002        |

    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 3                 |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.7   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |

    Given the average block duration is "1"

  @DiscTbl
  Scenario: Check the factors after each epoch, basically same as 0084-VDPR-012.
    Given the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR24 | ETH        | ETH   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | lp1    | ETH   | 10000000 |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR24 | 100000            | 0.02 | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR24 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR24 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/MAR24 | buy  | 100    | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/MAR24 | sell | 100    | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR24 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR24 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR24"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |
    And the market data for the market "ETH/MAR24" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 100000         | 1             |
    And the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0            | 0                | 0            |
      | party1 | 0            | 0                | 0            |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party3 | ETH/MAR24 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3 | ETH/MAR24 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
    When the network moves ahead "1" epochs
    Then the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0.007        | 0.006            | 0.005        |
      | party1 | 0            | 0                | 0            |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party3 | ETH/MAR24 | buy  | 20     | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3 | ETH/MAR24 | sell | 20     | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party1 | ETH/MAR24 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party1 | ETH/MAR24 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
    And the network moves ahead "1" epochs
    Then the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0.014        | 0.012            | 0.01         |
      | party1 | 0.007        | 0.006            | 0.005        |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |

    # when trade_value_for_fee_purposes>0, then total fee should be maker_fee_after_referral_discount+ treasury_fee + buyback_fee when fee_factor[infrastructure] = 0, fee_factor[liquidity] = 0 (0083-RFPR-053)
    # now lets reset the infra fee to 0 and do a trade with party 3:
    When the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.makerFee          | 0.1   |
      | market.fee.factors.infrastructureFee | 0     |
      | market.fee.factors.buybackFee        | 0.001 |
      | market.fee.factors.treasuryFee       | 0.002 |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party3 | ETH/MAR24 | sell | 11     | 0     | 1                | TYPE_MARKET | TIF_IOC |

    # trade value is 11*990 = 10,890
    # infra fee is set to 0
    # liquidity fee is set to 0
    # maker fee before discount = 10,890 * 0.1 => 1089
    # maker fee discount = 1089*0.014 => 15
    # maker fee after discount = 1089-15=1074
    # buyback = 11
    # treasury = 22
    # total = 1074 + 33
    Then the following trades should be executed:
      | seller | price | size | buyer | seller fee | seller infrastructure fee | seller liquidity fee | seller maker fee | seller infrastructure fee volume discount | seller liquidity fee volume discount | seller maker fee volume discount |
      | party3 | 990   | 11   | lp1   | 1107       | 0                         | 0                    | 1074             | 0                                         | 0                                    | 15                               |

    # when trade_value_for_fee_purposes>0, then total fee should be infrastructure_fee_after_referral_discount+ treasury_fee + buyback_fee when fee_factor[maker] = 0, fee_factor[liquidity] = 0 (0083-RFPR-055)
    # now lets reset the maker fee to 0 and do a trade with party 3:
    When the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.makerFee          | 0     |
      | market.fee.factors.infrastructureFee | 0.2   |

    # trade value is 11*990 = 10,890
    # maker fee is set to 0
    # liquidity fee is set to 0
    # infra fee before discount = 10,890 * 0.2 => 2178
    # infra fee discount = 2178*0.01 => 21
    # infra fee after discount = 2178-21=2157
    # buyback = 11
    # treasury = 22
    # total = 2157 + 33
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party3 | ETH/MAR24 | sell | 11     | 0     | 1                | TYPE_MARKET | TIF_IOC |

    Then the following trades should be executed:
      | seller | price | size | buyer | seller fee | seller infrastructure fee | seller liquidity fee | seller maker fee | seller infrastructure fee volume discount | seller liquidity fee volume discount | seller maker fee volume discount |
      | party3 | 990   | 11   | lp1   | 2190       | 2157                      | 0                    | 0                | 21                                        | 0                                    | 0                                |
    And the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0.014        | 0.012            | 0.01         |
      | party1 | 0.007        | 0.006            | 0.005        |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |

    # check if the tiers carry over in to the next epochs
    When the network moves ahead "1" epochs
    Then the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0.014        | 0.012            | 0.01         |
      | party1 | 0.007        | 0.006            | 0.005        |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |

    # 2 epochs later, nothing has changed
    When the network moves ahead "2" epochs
    Then the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0.014        | 0.012            | 0.01         |
      | party1 | 0.007        | 0.006            | 0.005        |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |

    # one epoch later, party 1 lost their benefits, party3 traded later on, they keep their benefits one more epoch.
    When the network moves ahead "1" epochs
    Then the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0.014        | 0.012            | 0.01         |
      | party1 | 0            | 0                | 0            |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |

    # next epoch, the benefits should have expired
    When the network moves ahead "1" epochs
    Then the parties have the following discount factors:
      | party  | maker factor | liquidity factor | infra factor |
      | party3 | 0            | 0                | 0            |
      | party1 | 0            | 0                | 0            |
      | party2 | 0            | 0                | 0            |
      | lp1    | 0            | 0                | 0            |
