Feature: Ensure the vAMM positions follow the market correctly

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

    # Setting up the accounts and vAMM submission now is part of the background, because we'll be running scenarios 0087-VAMM-006 through 0087-VAMM-014 on this setup
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
      | vamm1  | USD   | 1000    |

    When the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 600               | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 400               | 0.015 | submission |
    Then the network moves ahead "4" blocks
    And the current epoch is "0"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 70    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party3 | ETH/MAR22 | buy  | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 1      | 80    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party3 | ETH/MAR22 | sell | 10     | 151   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 10     | 152   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 39           | 1000           | 1             | 100       | 100       | 100              |
    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 1000   | 0.1      | 100  | 85          | 150         | 0.25               | 0.25               |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 1000   | STATUS_ACTIVE | 100  | 85          | 150         | 0.25               | 0.25               |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                             |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 1000   | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_LOW |

  @VAMM
  Scenario: 0087-VAMM-012: If other traders trade to move the market mid price to 90 and then in one trade move the mid price to 110 then trade to move the mid price back to 100 the vAMM will have a position of 0
    # to drop the mid price to 90, we need a sell order at 100 on the book
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 2      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 79           | 1000           | 2             | 100       | 90        | 90               |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party5 | 100   | 1    | party4 |

    # Now, in a single trade bump the mid price to 110, there is 1 sell order on tbe book for 1@100, place a buy order that uncrosses
    # and stays on the book, so volume should be 2
     When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 2      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 119          | 1000           | 3             | 100       | 110       | 110              |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party5 | 100   | 1    | party4 |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 159          | 1000           | 4             | 100       | 100       | 100              |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party5 | 100   | 1    | party4 |

    When the network moves ahead "1" blocks
    # we can't check the position outright, because the vAMM never opened a position
    # However, these positions cover the open interest, and we've accounted for all trades each step of the way
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 0              | 0            |        |
      | party2   | -1     | 0              | 0            |        |
      | party4   | -3     | 0              | 0            |        |
      | party5   | 3      | 0              | 0            |        |
      #| vamm1-id | 0      | 0              | 0            | true   |

  @VAMM
  Scenario: 0087-VAMM-014: If other traders trade to move the market mid price to 90 and then in one trade move the mid price to 110 then trade to move the mid price to 120 the vAMM will have a larger (more negative) but comparable position to if they had been moved straight from 100 to 120.
    # to drop the mid price to 90, we need a sell order at 100 on the book
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 2      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 79           | 1000           | 2             | 100       | 90        | 90               |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party5 | 100   | 1    | party4 |

    # Now, in a single trade bump the mid price to 110, there is 1 sell order on tbe book for 1@100, place a buy order that uncrosses
    # and stays on the book, so volume should be 2
     When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 2      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 119          | 1000           | 3             | 100       | 110       | 110              |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party5 | 100   | 1    | party4 |
    And debug detailed orderbook volumes for market "ETH/MAR22"
    
    # Now move the price to 120, we need some more buy and sell orders on the book, and we need to get rid of the sell 1@120
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/MAR22 | buy  | 3      | 120   | 2                | TYPE_LIMIT | TIF_GTC | p5b1      |
      | party4 | ETH/MAR22 | sell | 2      | 130   | 0                | TYPE_LIMIT | TIF_GTC | p4s       |
      | party5 | ETH/MAR22 | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | p5b2      |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 239          | 1000           | 6             | 100       | 120       | 120              |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 120   | 1    | party2   |        |
      | party5 | 113   | 2    | vamm1-id | true   |

    # check the PnL/position, we should see the vAMM holding short
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 20             | 0            |        |
      | party2   | -2     | -20            | 0            |        |
      | party4   | -2     | -40            | 0            |        |
      | party5   | 5      | 54             | 0            |        |
      | vamm1-id | -2     | -14            | 0            | true   |
