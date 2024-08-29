Feature: Derived key trades with its primary key.
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

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    # Create 2 identical markets, one will be used to test moving the mid price in steps of one, the other will do the same in a single trade.
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    # Setting up the accounts and vAMM submission now is part of the background, because we'll be running scenarios 0090-VAMM-006 through 0090-VAMM-014 on this setup
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | USD   | 1000000 |
      | lp2    | USD   | 1000000 |
      | lp3    | USD   | 1000000 |
      | lp4    | USD   | 1000000 |
      | party1 | USD   | 1000000 |
      | party2 | USD   | 1000000 |
      | party3 | USD   | 1000000 |
      | party4 | USD   | 1000000 |
      | party5 | USD   | 1000000 |
      | party6 | USD   | 1000000 |
      | vamm1  | USD   | 1000000 |
      | vamm2  | USD   | 1000000 |

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
      | vamm1 | ETH/MAR22 | 100000 | 0.1      | 100  | 85          | 150         | 4              | 4              | 0.01         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 85          | 150         | 4              | 4              |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |

  @VAMM
  Scenario: Simply have the vamm1 submit an order to the book that uncrosses with its own derived key.
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | vamm1 | ETH/MAR22 | buy  | 1      | 101   | 1                | TYPE_LIMIT | TIF_GTC | vamm1-b   |
    And the network moves ahead "1" blocks
    # trade with own derived key
    Then the following trades should be executed:
      | buyer | price | size | seller   | is amm |
      | vamm1 | 100   | 1    | vamm1-id | true   |
	And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 0              | 0            |        |
      | party2   | -1     | 0              | 0            |        |
      | vamm1    | 1      | 0              | 0            |        |
      | vamm1-id | -1     | 0              | 0            | true   |

    # Now assume someone managed to submit an order on behalf of the derived key
    When the parties place the following hacked orders:
      | party    | market id | side | volume | price | resulting trades | type        | tif     | reference | is amm |
      | vamm1-id | ETH/MAR22 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_FOK | vamm-b    | true   |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 160   | 1    | lp1    | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 60             | 0            |        |
      | party2   | -1     | -60            | 0            |        |
      | vamm1    | 1      | 60             | 0            |        |
      | vamm1-id | 0      | 0              | -60          | true   |
      | lp1      | -1     | 0              | 0            |        |

    # let's re-open the position for the vAMM, and cancel it using the reduce only method
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | vamm1 | ETH/MAR22 | buy  | 2      | 101   | 1                | TYPE_LIMIT | TIF_GTC | vamm1-b2  |
    Then the following trades should be executed:
      | buyer | price | size | seller   | is amm |
      | vamm1 | 100   | 2    | vamm1-id | true   |

    # Check the positions
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 0              | 0            |        |
      | party2   | -1     | 0              | 0            |        |
      | vamm1    | 3      | 0              | 0            |        |
      | vamm1-id | -2     | 0              | -60          | true   |
      | lp1      | -1     | 60             | 0            |        |

    # Now the vamm shouldn't generate any more sell orders
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm1 | ETH/MAR22 | METHOD_REDUCE_ONLY |
    # ensure no sell trades
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC | p1-b2     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | sell | 1      | 90    | 1                | TYPE_LIMIT | TIF_GTC | p2-s2     |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 100   | 1    | party2 | true   |

    # ensure the vAMM position is indeed reduced
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 0              | 0            |        |
      | party2   | -2     | 0              | 0            |        |
      | vamm1    | 3      | 0              | 0            |        |
      | vamm1-id | -1     | 0              | -60          | true   |
      | lp1      | -1     | 60             | 0            |        |

    # Now let's see what happens if someone manages to submit a sell order for a reduce-only AMM key
    When the parties place the following hacked orders:
      | party    | market id | side | volume | price | resulting trades | type        | tif     | reference | is amm |
      | vamm1-id | ETH/MAR22 | buy  | 2      | 0     | 1                | TYPE_MARKET | TIF_FOK | vamm-c    | true   |
    # indeed, the order the vAMM should never create is accepted, and gets executed.
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 160   | 2    | lp1    | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 60             | 0            |        |
      | party2   | -2     | -120           | 0            |        |
      | vamm1    | 3      | 180            | 0            |        |
      | vamm1-id | 1      | 0              | -120         | true   |
      | lp1      | -3     | 0              | 0            |        |

    # Now the vAMM has switched to long, so it should not trade with a sell order.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | sell | 1      | 90    | 0                | TYPE_LIMIT | TIF_GTC | p2-s3     |

    # But it'll use the buy order to close its own position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 2      | 100   | 2                | TYPE_LIMIT | TIF_GTC | p1-b3     |
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party1 | 90    | 1    | party2   |        |
      | party1 | 99    | 1    | vamm1-id | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 3      | 8              | 0            |        |
      | party2   | -3     | -7             | 0            |        |
      | vamm1    | 3      | -3             | 0            |        |
      | vamm1-id | 0      | 0              | -181         | true   |
      | lp1      | -3     | 183            | 0            |        |
    # The AMM pool is indeed cancelled.
    And the AMM pool status should be:
      | party | market id | amount | status           | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_CANCELLED | 100  | 85          | 150         | 4              | 4              |

    # Trying to place a vAMM order again results in a margin check failure (the accounts have been drained)
    When the parties place the following hacked orders:
      | party    | market id | side | volume | price | resulting trades | type        | tif     | reference | is amm | error               |
      | vamm1-id | ETH/MAR22 | buy  | 2      | 0     | 0                | TYPE_MARKET | TIF_FOK | vamm-d    | true   | margin check failed |
