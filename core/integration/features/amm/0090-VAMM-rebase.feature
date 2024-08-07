Feature: vAMM rebasing when created or amended

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
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    # Setting up the accounts and vAMM submission now is part of the background, because we'll be running scenarios 0090-VAMM-006 through 0090-VAMM-014 on this setup
    And the parties deposit on asset's general account the following amount:
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
      | lp_1 | lp1   | ETH/MAR22 | 10000             | 0.02 | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |

    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |
      
    Then the network moves ahead "1" epochs
    And the current epoch is "1"

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.05     | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         |

  @VAMM
  Scenario: a vAMM submits a rebasing order to SELL when its base is lower than an existing AMM's (0090-VAMM-033)

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000 | 0.05     | 95   | 90          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 95   | 90          | 105         |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |

    # second AMM has its base 5 away from the first AMM so it must submit a rebasing-order 
    And the following trades should be executed:
      | buyer    | price | size | seller   | is amm |
      | vamm1-id | 98    | 140  | vamm2-id | true   |
    Then the network moves ahead "1" blocks

    # and now the mid-price has shifted lower to a value between the two AMM's bases 95 < 97 < 100
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price |
      | 98         | TRADING_MODE_CONTINUOUS | 97        |


  @VAMM
  Scenario: a vAMM submission cannot rebase because it exceeds slippage but exceeds slippage

    # sell rebase order
    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee | error                                  |
      | vamm2 | ETH/MAR22 | 100000 | 0.0005   | 95   | 90          | 105         | 0.03         | not enough liquidity for AMM to rebase |
    Then the AMM pool status should be:
      | party | market id | amount | status          | base | lower bound | upper bound | reason                      |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_REJECTED | 95   | 90          | 105         | STATUS_REASON_CANNOT_REBASE |
    
    # buy rebase order
    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee | error                                  |
      | vamm2 | ETH/MAR22 | 100000 | 0.0005   | 105  | 100         | 110         | 0.03         | not enough liquidity for AMM to rebase |
    Then the AMM pool status should be:
      | party | market id | amount | status          | base | lower bound | upper bound | reason                      |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_REJECTED | 105  | 100         | 110         | STATUS_REASON_CANNOT_REBASE |

  @VAMM
  Scenario: a vAMM submits a rebasing order to BUY when its base is lower than an existing AMM's (0090-VAMM-033)

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000 | 0.05     | 105  | 100         | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 105  | 100         | 110         |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |

    # second AMM has its base 5 away from the first AMM so it must submit a rebasing-order 
    And the following trades should be executed:
      | buyer    | price | size | seller   | is amm |
      | vamm2-id | 101   | 176  | vamm1-id | true   |
    Then the network moves ahead "1" blocks

    # and now the mid-price has shifted lower to a value between the two AMM's bases 100 < 104 < 105
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | 
      | 101        | TRADING_MODE_CONTINUOUS | 103       |
  


  @VAMM
  Scenario: two aligned AMM's and one amends shifting its base lower and needs to submit a SELL rebasing order

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound |  proposed fee |
      | vamm2 | ETH/MAR22 | 100000 | 0.05     | 100  | 95          | 105         |  0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 95          | 105         |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |

    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | 
      | 100        | TRADING_MODE_CONTINUOUS | 100       |

    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 0.1      | 95   | 90          | 105         | 
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 95   | 90          | 105         |


    # second AMM has its base 5 away from the first AMM so it must submit a rebasing-order 
    And the following trades should be executed:
      | buyer    | price | size | seller   | is amm |
      | vamm1-id | 98    | 140  | vamm2-id | true   |
    Then the network moves ahead "1" blocks

    # and now the mid-price has shifted lower to a value between the two AMM's bases 95 < 98 < 100
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price |
      | 98         | TRADING_MODE_CONTINUOUS | 97        |


  @VAMM
  Scenario: two aligned AMM's and one amends shifting its base lower and needs to submit a BUY rebasing order

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000 | 0.05     | 100  | 95          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 95          | 105         |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |

    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | 
      | 100        | TRADING_MODE_CONTINUOUS | 100       |

    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 0.1      | 105  | 100         | 110         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 105  | 100         | 110         |


    # second AMM has its base 5 away from the first AMM so it must submit a rebasing-order 
    And the following trades should be executed:
      | buyer    | price | size | seller   | is amm |
      | vamm2-id | 101   | 176  | vamm1-id | true   |
    Then the network moves ahead "1" blocks

    # and now the mid-price has shifted lower to a value between the two AMM's bases 100 < 104 < 105
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | 
      | 101        | TRADING_MODE_CONTINUOUS | 103       |


  @VAMM
  Scenario: One AMM exists and another on is submitted such that their ranges are disjoint and cross entirely (0090-VAMM-034)

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000 | 0.50     | 200  | 195         | 205         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 200  | 195         | 205         |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |

    Then the network moves ahead "1" blocks

    # second AMM has its base 5 away from the first AMM so it must submit a rebasing-order 
    And the following trades should be executed:
      | buyer    | price | size | seller   | is amm |
      | vamm2-id | 102   | 262  | vamm1-id | true   |

    Then the network moves ahead "1" blocks

    # and now the mid-price has shifted lower to a value between the two AMM's bases 100 < 104 < 105
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price |
      | 102        | TRADING_MODE_CONTINUOUS | 107       |
