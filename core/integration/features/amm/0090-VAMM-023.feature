Feature: Test cancelled vAMM becomes active on amend.

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
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.1   |
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
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.95        | 3                 |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

  @VAMM
  Scenario: 0090-VAMM-023: If a vAMM is cancelled and set into Reduce-Only mode, then an amend is sent by the user who created it, the vAMM is amended according to those instructions and is moved out of Reduce-Only mode back into normal operation.
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | lp3    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |
      | vamm1  | USD   | 100000 |

    When the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 600               | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 400               | 0.015 | submission |
    Then the network moves ahead "4" blocks
    And the current epoch is "0"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/MAR22 | buy  | 10     | 85    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 10     | 90    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party3 | ETH/MAR22 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | buy  | 10     | 95    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | sell | 10     | 106   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS  | 39           | 1000           | 1             | 100       | 100       | 100              |

    # trade to move the mark price to 105
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 105   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 10     | 105   | 1                | TYPE_LIMIT | TIF_GTC |           |

    # Now submit our vAMM, no trades should happen
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.1      | 100  | 85          | 150         | 0.25               | 0.25               | 0.01         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 85          | 150         | 0.25               | 0.25               |

    And set the following AMM sub account aliases:
      | party | market id | alias     |
      | vamm1 | ETH/MAR22 | vamm1-acc |
    And the following transfers should happen:
      | from  | from account         | to        | to account           | market id | amount | asset | is amm | type                             |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-acc | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_LOW |
    And the parties should have the following account balances:
      | party     | asset | market id | general | margin | is amm |
      | vamm1     | USD   |           | 0       |        |        |
      | vamm1-acc | USD   | ETH/MAR22 | 100000  |        | true   |

    # Now cancel with reduce-only
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm1 | ETH/MAR22 | METHOD_REDUCE_ONLY |
    Then the AMM pool status should be:
      | party | market id | amount | status             | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_REDUCE_ONLY | 100  | 85          | 150         | 0.25               | 0.25               |
    # Balance is not yet released
    And the parties should have the following account balances:
      | party     | asset | market id | general | margin | is amm |
      | vamm1     | USD   |           | 0       |        |        |
      | vamm1-acc | USD   | ETH/MAR22 | 100000  |        | true   |

    # Now amend the reduce-only vAMM submission, and check to see if its status returns back to active
    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 0.1      | 105  | 90          | 155         | 0.25           | 0.25           |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 105  | 90          | 155         | 0.25           | 0.25               |

    # Now trigger a MTM settlement, this should not change anything, whereas without the amend it'd move the vAMM to the cancelled status.
    When the network moves ahead "1" blocks
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 105  | 90          | 155         | 0.25           | 0.25           |
    And the parties should have the following account balances:
      | party     | asset | market id | general | margin | is amm |
      | vamm1     | USD   |           | 0       |        |        |
      | vamm1-acc | USD   | ETH/MAR22 | 100000  |        | true   |
    # The mark price has now moved to 105
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 105        | TRADING_MODE_CONTINUOUS | 461          | 1000           | 11            | 104       | 105       | 105              |
