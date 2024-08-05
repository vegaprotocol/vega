  Feature: Attempt to replicate a bug in vAMM

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
      | vamm1  | USD   | 1000000 |

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

  @VAMMP
  Scenario: AMM holds a long position, there should be no more buy orders created, when submitting a sell order at a lower price, a panic seemingly can occur.
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 100       | 100              | 101              | 99             |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 500    | 155   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 122   | 291  | vamm1-id | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 157       | 157              | 160              | 155            |

    # trying to trade again causes no trades because the AMM has no more volume
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 500    | 150     | 0                | TYPE_LIMIT | TIF_GTC |

    # the AMM's mid price has moved to 150, but it has no volume +150 so that best offer comes from the orderbook of 160
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 157       | 157              | 160              | 155            |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 291    | 0              | 0            |        |
      | vamm1-id | -291   | 0              | 0            | true   |
    # Notional value therefore is 317 * 122
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 122        | TRADING_MODE_CONTINUOUS | 157       | 157              | 160              | 155            |

    # vAMM receives fees, but loses out in the MTM settlement
    And the following transfers should happen:
       | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 143    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN     | ETH/MAR22 | 74548  | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 1      | 160   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 160   | 1    | lp1    | false  |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 291    | 11058          | 0            |        |
      | party5   | 1      | 0              | 0            |        |
      | lp1      | -1     | 0              | 0            |        |
      | vamm1-id | -291   | -11058         | 0            | true   |

    # Now the same buy use a market order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party5 | ETH/MAR22 | buy  | 2      | 0     | 1                | TYPE_MARKET | TIF_FOK |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 160   | 2    | lp1    | false  |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 291    | 11058          | 0            |        |
      | party5   | 3      | 0              | 0            |        |
      | lp1      | -3     | 0              | 0            |        |
      | vamm1-id | -291   | -11058         | 0            | true   |

    When the parties cancel the following AMM:
      | party | market id | method           |
      | vamm1 | ETH/MAR22 | METHOD_IMMEDIATE |
    Then the AMM pool status should be:
      | party | market id | amount | status           | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_CANCELLED | 100  | 85          | 150         | 4              | 4              |
    And the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | lower leverage | proposed fee |
      | vamm1 | ETH/MAR22 | 99999  | 0.1      | 100  | 85          | 4              | 0.01         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | lower leverage |
      | vamm1 | ETH/MAR22 | 99999  | STATUS_ACTIVE | 100  | 85          | 4              |
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm2-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm2-id | ACCOUNT_TYPE_GENERAL |           | 99999  | USD   | true   | TRANSFER_TYPE_AMM_LOW |
    ## This is what we're after
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 291    | 11058          | 0            |        |
      | party1   | 1      | 60             | 0            |        |
      | party5   | 3      | 0              | 0            |        |
      | lp1      | -3     | 0              | 0            |        |
      | vamm1-id | 0      | 0              | -11058       | true   |

    # Now the same but use a sell order that is outside of the range
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | sell | 1      | 150   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party4 | 155   | 1    | party1 | false  |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 292    | 9603           | 0            |        |
      | party1   | 0      | 0              | 55           |        |
      | party5   | 3      | -15            | 0            |        |
      | lp1      | -3     | 15             | 0            |        |
      | vamm1-id | 0      | 0              | -11058       | true   |

    When the network moves ahead "1" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 100    | 90    | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC |
