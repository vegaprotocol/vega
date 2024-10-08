Feature: Test vAMM implied commitment is working as expected

  Background:
    Given the average block duration is "1"
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    And the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau                   | mu | r   | sigma |
      | 0.001         | 0.0011407711613050422 | 0  | 0.9 | 3.0   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.00             | 20s         | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 60s   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 6     |
      | market.auction.minimumDuration                      | 1     |
      | market.fee.factors.infrastructureFee                | 0.001 |
      | market.fee.factors.makerFee                         | 0.004 |
      | spam.protection.max.stopOrdersPerMarket             | 5     |
      | market.liquidity.equityLikeShareFeeFraction         | 1     |
      | market.amm.minCommitmentQuantum                     | 1     |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0     |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.25  |
      | market.liquidity.providersFeeCalculationTimeStep    | 9s    |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the following assets are registered:
      | id  | decimal places |
      | USD | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.05        | 1                            | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | USD   | 1000000 |
      | lp2    | USD   | 1000000 |
      | lp3    | USD   | 1000000 |
      | party1 | USD   | 1000000 |
      | party2 | USD   | 1000000 |
      | party3 | USD   | 1000000 |
      | party4 | USD   | 1000000 |
      | party5 | USD   | 1000000 |
      | vamm1  | USD   | 1000000 |
      | vamm2  | USD   | 1000000 |

    When the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 10000            | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 10000            | 0.02 | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 20     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |

    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |

    Then the network moves ahead "1" epochs
    And the current epoch is "1"

  @VAMM
  Scenario: 0042-LIQF-108: A vAMM which was active on the market with an average of `10000` liquidity units (`price * volume`) provided across the epoch, and where the `market.liquidity.stakeToCcyVolume` value is `100`, will have an implied commitment of `100`.
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee | 
      | vamm1 | ETH/MAR22 | 10000 | 0.05      | 100  | 98          | 102         | 0.03         | 
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 10000 | STATUS_ACTIVE  | 100  | 98          | 102         | 

    And set the following AMM sub account aliases:
      | party | market id | alias      |
      | vamm1 | ETH/MAR22 | vamm-party |

    # Then the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party2 | ETH/MAR22 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 10   | party2 |

    Then the network moves ahead "1" epochs
    And the current epoch is "2"

    And the following transfers should happen:
      | type                                           | from   | to                                                               | from account                                   | to account                                     | market id | amount | asset |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 3      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp1                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 8      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp2                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 8      | USD   |
      | TRANSFER_TYPE_SLA_PERFORMANCE_BONUS_DISTRIBUTE | market | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/MAR22 | 16     | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_UNPAID_COLLECT     | lp1    | market                                                           | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/MAR22 | 8      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_UNPAID_COLLECT     | lp2    | market                                                           | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/MAR22 | 8      | USD   |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
    | party                                                            | equity like share  | virtual stake         | average entry valuation |
    | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 0                  | 4669.0000000000000000 | 24669                   |

  @VAMM
  Scenario: 0042-LIQF-110: A vAMM which was active on the market with an average of `10000` liquidity units (`price * volume`) provided for half the epoch, because it was submitted half way through an epoch, and where the `market.liquidity.stakeToCcyVolume` value is `100`, will have an implied commitment of `50`.

    # move ahead half an epoch then submit the AMM
    Then the network moves ahead "5" blocks
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 10000  | 0.05     | 100  | 98          | 102         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 10000  | STATUS_ACTIVE | 100  | 98          | 102         |


    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party2 | ETH/MAR22 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC |           |

    # epoch ends and AMM should get fewer fees than the above test because its ELS is less
    Then the network moves ahead "7" blocks
    And the following transfers should happen:
      | type                                           | from   | to                                                               | from account                                   | to account                                     | market id | amount | asset |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 0      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp1                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 9      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp2                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 9      | USD   |

    # virtual stake is HALF what it was in the above test since the AMM was only active for half the epoch
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
    | party                                                            | equity like share  | virtual stake         | average entry valuation |
    | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 0                  | 2334.0000000000000000 | 22334                   |


  @VAMM
  Scenario: LP fee for AMM based solely on liquidity score

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.equityLikeShareFeeFraction         | 0     |


    # submit an AMM
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 10000  | 0.05     | 100  | 98          | 102         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 10000  | STATUS_ACTIVE | 100  | 98          | 102         |

     Then the network moves ahead "1" blocks

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party2 | ETH/MAR22 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs
    And the following transfers should happen:
      | type                                           | from   | to                                                               | from account                                   | to account                                     | market id | amount | asset |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 6      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp1                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 6      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp2                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 6      | USD   |


  @VAMM
  Scenario: LP fee for AMM based solely on liquidity score when AMM cancels half way through an epoch

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.equityLikeShareFeeFraction         | 0     |

    # submit an AMM
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 10000  | 0.05     | 100  | 98          | 102         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 10000  | STATUS_ACTIVE | 100  | 98          | 102         |

     Then the network moves ahead "5" blocks
     When the parties cancel the following AMM:
     | party  | market id | method           |
     | vamm1 | ETH/MAR22 | METHOD_IMMEDIATE |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party2 | ETH/MAR22 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC |           |

    # epoch ends and AMM should get fewer fees than the above test because its ELS is less
    Then the network moves ahead "7" blocks
    And the following transfers should happen:
      | type                                           | from   | to                                                               | from account                                   | to account                                     | market id | amount | asset |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 5      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp1                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 7      | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE           | market | lp2                                                              | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 7      | USD   |

    # and at the next epoch it should be 0 fees because its fully cancelled
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party2 | ETH/MAR22 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs
    And the following transfers should happen:
      | type                                   | from   | to    | from account                           | to account                       | market id | amount | asset |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE   | market | lp1   | ACCOUNT_TYPE_FEES_LIQUIDITY            | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | ETH/MAR22 | 10     | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE   | market | lp2   | ACCOUNT_TYPE_FEES_LIQUIDITY            | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | ETH/MAR22 | 10     | USD   |