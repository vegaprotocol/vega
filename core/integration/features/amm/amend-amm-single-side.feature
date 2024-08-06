Feature: vAMM amend single-sided commitments

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
      | ETH/MAR23 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    # Setting up the accounts and vAMM submission now is part of the background, because we'll be running scenarios 0090-VAMM-006 through 0090-VAMM-014 on this setup
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
      | lp_1 | lp1   | ETH/MAR22 | 10000             | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR23 | 10000             | 0.02 | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp2    | ETH/MAR23 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party3 | ETH/MAR23 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party4 | ETH/MAR23 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |
      | lp2    | ETH/MAR23 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp2-s     |

    # End opening auction for both markets.
    When the network moves ahead "3" blocks
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |
      | party3 | 100   | 1    | party4 |
      
    Then the network moves ahead "1" epochs
    And the current epoch is "1"

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.05     | 100  | 110         | 0.03         |
    And the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR23 | 100000 | 0.05     | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 110         |             |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 110         | 90          |
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR23 | vamm2-id |


  @VAMM
  Scenario: vAMM is amended from BUY to SELL side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 40    | 20     |
      | buy  | 99    | 1      |
      | sell | 160   | 10     |
    And the parties amend the following AMM:
      | party | market id | slippage | base | lower bound |
      | vamm1 | ETH/MAR22 | 0.05     | 100  | 90          |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          |             |
    # Make sure nothing changes when we move to the next block
    When the network moves ahead "1" blocks
    And the parties amend the following AMM:
      | party | market id | slippage | base | lower bound |
      | vamm1 | ETH/MAR22 | 0.05     | 100  | 90          |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          |             |
    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 40    | 20     |
      | buy  | 99    | 1      |
      | sell | 160   | 10     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 1      | 101   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          |             |
    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 40    | 20     |
      | buy  | 99    | 1      |
      | sell | 101   | 1      |
      | sell | 160   | 10     |
    When the network moves ahead "1" blocks
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          |             |
    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 40    | 20     |
      | buy  | 99    | 1      |
      | sell | 101   | 1      |
      | sell | 160   | 10     |

  @VAMM
  Scenario: vAMM is amended from having a lower and upper bound to just lower or upper bound. The vAMM does not hold any positions.
    When the parties amend the following AMM:
      | party | market id | slippage | base | upper bound |
      | vamm2 | ETH/MAR23 | 0.05     | 100  | 110         |
    # No trades because the sell vAMM places no sell orders.
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR23 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 110         |             |

    When the network moves ahead "1" blocks
    And the parties amend the following AMM:
      | party | market id | slippage | base | lower bound |
      | vamm2 | ETH/MAR23 | 0.05     | 100  | 90          |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 90          |             |
    And the order book should have the following volumes for market "ETH/MAR23":
      | side | price | volume |
      | buy  | 40    | 20     |
      | buy  | 99    | 1      |
      | sell | 160   | 10     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR23 | sell | 1      | 101   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 90          |             |
    And the order book should have the following volumes for market "ETH/MAR23":
      | side | price | volume |
      | buy  | 40    | 20     |
      | buy  | 99    | 1      |
      | sell | 101   | 1      |
      | sell | 160   | 10     |
    When the network moves ahead "1" blocks
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 90          |             |
    And the order book should have the following volumes for market "ETH/MAR23":
      | side | price | volume |
      | buy  | 40    | 20     |
      | buy  | 99    | 1      |
      | sell | 101   | 1      |
      | sell | 160   | 10     |

  @VAMM
  Scenario: vAMM can't amended from having a lower and upper bound to just lower or upper bound when the vAMM holds a long position.
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR23 | buy  | 1      | 101   | 1                | TYPE_LIMIT | TIF_GTC |
    When the parties amend the following AMM:
      | party | market id | slippage | base | upper bound |
      | vamm2 | ETH/MAR23 | 0.05     | 100  | 110         |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 110         |             |

    When the network moves ahead "1" blocks
    Then the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | error                                       |
      | vamm2 | ETH/MAR23 | 0.05     | 100  | 90          | cannot remove upper bound when AMM is short |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 110         |             |

  @VAMM
  Scenario: vAMM can't amended from having a lower and upper bound to just lower or upper bound when the vAMM holds a short position.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR23 | sell | 1      | 99    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties amend the following AMM:
      | party | market id | slippage | base | lower bound |
      | vamm2 | ETH/MAR23 | 0.05     | 100  | 90          |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  |             | 90          |

    When the network moves ahead "1" blocks
    Then the parties amend the following AMM:
      | party | market id | slippage | base | upper bound | error                                      |
      | vamm2 | ETH/MAR23 | 0.05     | 100  | 110         | cannot remove lower bound when AMM is long |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  |             | 90          |

  @VAMM
  Scenario: vAMM can't amended from having a single bound to the other side, when the vAMM holds a long position.
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 101   | 1                | TYPE_LIMIT | TIF_GTC |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 110         |             |

    When the network moves ahead "1" blocks
    Then the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | error                                       |
      | vamm1 | ETH/MAR22 | 0.05     | 100  | 90          | cannot remove upper bound when AMM is short |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 110         |             |

  @VAMM
  Scenario: vAMM can't amended from having a single bound to the other side, when the vAMM holds a short position.
    # No trades because the sell vAMM places no sell orders.
    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound |
      | vamm1 | ETH/MAR22 | 0.05     | 100  | 90          |
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/MAR22 | sell | 1      | 99    | 1                | TYPE_LIMIT | TIF_GTC |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  |             | 90          |

    When the network moves ahead "1" blocks
    Then the parties amend the following AMM:
      | party | market id | slippage | base | upper bound | error                                      |
      | vamm1 | ETH/MAR22 | 0.05     | 100  | 110         | cannot remove lower bound when AMM is long |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound | lower bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  |             | 90          |
