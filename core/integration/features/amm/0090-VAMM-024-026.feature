Feature: When market.amm.minCommitmentQuantum is 1000, mid price of the market 100, and a user with 1000 USDT creates a vAMM with commitment 1000, base price 100, upper price 150, lower price 85 and leverage ratio at each bound 0.25

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
	  | market.amm.minCommitmentQuantum                     | 1000  |
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

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

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
      | vamm1  | USD   | 30000   |

    When the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 600               | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 400               | 0.015 | submission |
    Then the network moves ahead "4" blocks
    And the current epoch is "0"

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

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 39           | 1000           | 1             | 100       | 100       | 100              |
    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | proposed fee |
      | vamm1 | ETH/MAR22 | 30000  | 0.1      | 100  | 85          | 150         | 4              | 4              | 0.01         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 30000  | STATUS_ACTIVE | 100  | 85          | 150         | 4              | 4              |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 30000  | USD   | true   | TRANSFER_TYPE_AMM_LOW |

  @VAMM
  Scenario: 0090-VAMM-024: If other traders trade to move the market mid price to 140 the vAMM has a short position.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 4      | 120   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short

    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 100   | 4    | vamm1-id | true   |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 102       | 102              |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 4      | 0              | 0            |        |
      | vamm1-id | -4     | 0              | 0            | true   |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 30000  | STATUS_ACTIVE | 100  | 85          | 150         | 4              | 4              |

  @VAMM
  Scenario: 0090-VAMM-025: If the vAMM is then amended such that it has a new base price of 140 it should attempt to place a trade to rebalance it's position to 0 at a mid price of 140. If that trade can execute with the slippage as configured in the request then the transaction is accepted.
    # other parties place large order between AMM base of 100 and 140
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/MAR22 | sell | 50     | 120   | 0                | TYPE_LIMIT | TIF_GTC |           |

    # Now amend the vAMM and see its rebasing trade happen
    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | upper bound |  lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 0.5      | 140  | 90          | 155         |  4              | 4              |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 30000  | STATUS_ACTIVE | 140  | 90          | 155         | 4              | 4              |


    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 120   | 49   | party4 | true   |
    # ensure the vamm closed its position
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price |
      | 120        | TRADING_MODE_CONTINUOUS | 119       | 119              |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -49    | 0              | 0            |        |
      | vamm1-id | 49     | 0              | 0            | true   |

  @VAMM
  Scenario: 0090-VAMM-026: If the trade cannot execute with the slippage as configured in the request then the transaction is rejected and no changes to the vAMM are made.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 4      | 120   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 100   | 4    | vamm1-id | true   |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 102       | 102              |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 4      | 0              | 0            |        |
      | vamm1-id | -4     | 0              | 0            | true   |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 30000  | STATUS_ACTIVE | 100  | 85          | 150         | 4              | 4              |

    # Now amend the vAMM such that there won't be enough liquidity to rebase
    When the parties amend the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | error                                  |
      | vamm1 | ETH/MAR22 | 10005  | 0.01     | 165  | 160         | 170         | not enough liquidity for AMM to rebase |
    # ensure the status of the vAMM remains the same (if no update event is sent, this test will pass even if the vAMM was in some way changed)
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 30000  | STATUS_ACTIVE | 100  | 85          | 150         | 4              | 4              |

    # check that the account balances have not been updated either
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1-id | USD   | ETH/MAR22 | 29162   | 840    | true   |

    # To account for a passing test caused by an event not being sent out, cancel the vAMM and check the status
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm1 | ETH/MAR22 | METHOD_REDUCE_ONLY |
    Then the AMM pool status should be:
      | party | market id | amount | status             | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 30000  | STATUS_REDUCE_ONLY | 100  | 85          | 150         | 4              | 4              |
