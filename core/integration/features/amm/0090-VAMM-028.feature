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

    # Create 2 identical markets, one will be used to test moving the mid price in steps of one, the other will do the same in a single trade.
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
      | lp_3 | lp3   | ETH/MAR23 | 600               | 0.02  | submission |
      | lp_4 | lp4   | ETH/MAR23 | 400               | 0.015 | submission |
    Then the network moves ahead "4" blocks
    And the current epoch is "0"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp3    | ETH/MAR23 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp3-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party3 | ETH/MAR23 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party4 | ETH/MAR23 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |
      | lp3    | ETH/MAR23 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp3-s     |
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |
      | party3 | 100   | 1    | party4 |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 39           | 1000           | 1             | 100       | 100       | 100              |
    And the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 39           | 1000           | 1             | 100       | 100       | 100              |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.1      | 100  | 85          | 150         | 4              | 4              | 0.01         |
      | vamm2 | ETH/MAR23 | 100000 | 0.1      | 100  | 85          | 150         | 4              | 4              | 0.01         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 85          | 150         | 4              | 4              |
      | vamm2 | ETH/MAR23 | 100000 | STATUS_ACTIVE | 100  | 85          | 150         | 4              | 4              |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR23 | vamm2-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |
      | vamm2 | ACCOUNT_TYPE_GENERAL | vamm2-id | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |

  @VAMM
  Scenario: 0090-VAMM-028: The volume quoted to move from price 100 to price 110 in one step is the same as the sum of the volumes to move in 10 steps of 1.
    # Move mid price to 110 in one go. A volume of 74 is the minimum required, with 73 we only get to 109
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 74     | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2999         | 1000           | 75            | 100       | 110       | 110              | 111              | 109            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 104   | 74   | vamm1-id | true   |
    # Check vAMM position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party5   | 74     | 0              | 0            |        |
      | vamm1-id | -74    | 0              | 0            | true   |

    # Now do the same thing as above, only for ETH/MAR23
    # Move mid price to 101
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 79           | 1000           | 2             | 100       | 101       | 101              | 102              | 100            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 100   | 1    | vamm2-id | true   |

    # Move mid price to 102
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 399          | 1000           | 10            | 100       | 102       | 102              | 103              | 101            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 101   | 8    | vamm2-id | true   |

    # Move mid price to 103
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 9      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 759          | 1000           | 19            | 100       | 103       | 103              | 104              | 102            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 102   | 9    | vamm2-id | true   |

    # Move mid price to 104
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 1079         | 1000           | 27            | 100       | 104       | 104              | 105              | 103            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 103   | 8    | vamm2-id | true   |

    # Move mid price to 105
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 1399         | 1000           | 35            | 100       | 105       | 105              | 106              | 104            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 104   | 8    | vamm2-id | true   |

    # Move mid price to 106
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 1719         | 1000           | 43            | 100       | 106       | 106              | 107              | 105            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 105   | 8    | vamm2-id | true   |

    # Move mid price to 107
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2039         | 1000           | 51            | 100       | 107       | 107              | 108              | 106            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 106   | 8    | vamm2-id | true   |

    # Move mid price to 108
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2359         | 1000           | 59            | 100       | 108       | 108              | 109              | 107            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 107   | 8    | vamm2-id | true   |

    # Move mid price to 109
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2679         | 1000           | 67            | 100       | 109       | 109              | 110              | 108            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 108   | 8    | vamm2-id | true   |

    # Finally, move to 110, the volume should be the same, so open interest should be 75 -> + 8
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | buy  | 8      | 111   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2999         | 1000           | 75            | 100       | 110       | 110              | 111              | 109            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party6 | 109   | 8    | vamm2-id | true   |
    
    # Confirm the volume matches what we expect, but note the PnL can differ as we have multiple trades at different price-points.
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party3   | 1      | 9              | 0            |        |
      | party4   | -1     | -9             | 0            |        |
      | party6   | 74     | 304            | 0            |        |
      | vamm2-id | -74    | -304           | 0            | true   |
      | vamm1-id | -74    | 0              | 0            | true   |

  @VAMM
  Scenario: 0090-VAMM-029: The volume quoted to move from price 100 to price 90 in one step is the same as the sum of the volumes to move in 10 steps of 1.
    # Move mid price to 90 in one go. A volume of 347 is the minimum required, 346 only gets us to 91
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 347    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13915        | 1000           | 348           | 100       | 90        | 90               | 91               | 89             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 95    | 347  | party5 | true   |
    # Check vAMM position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -5             | 0            |        |
      | party2   | -1     | 5              | 0            |        |
      | party5   | -347   | 0              | 0            |        |
      | vamm1-id | 347    | 0              | 0            | true   |

    # Now do the same thing as above, only for ETH/MAR23
    # Move mid price to 99
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 79           | 1000           | 2             | 100       | 99        | 99               | 100              | 98             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 99    | 1    | party6 | true   |

    # Move mid price to 98
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 36     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 1519         | 1000           | 38            | 100       | 98        | 98               | 99               | 97             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 98    | 36   | party6 | true   |

    # Move mid price to 97
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 36     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2959         | 1000           | 74            | 100       | 97        | 97               | 98               | 96             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 97    | 36   | party6 | true   |

    # Move mid price to 96
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 38     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 4478         | 1000           | 112           | 100       | 96        | 96               | 97               | 95             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 96    | 38   | party6 | true   |

    # Move mid price to 95
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 37     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 5958         | 1000           | 149           | 100       | 95        | 95               | 96               | 94             |
    And debug trades
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 95    | 37   | party6 | true   |

    # Move mid price to 94
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 39     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 7517         | 1000           | 188           | 100       | 94        | 94               | 95               | 93             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 94    | 39   | party6 | true   |

    # Move mid price to 93
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 39     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 9077         | 1000           | 227           | 100       | 93        | 93               | 94               | 92             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 93    | 39   | party6 | true   |

    # Move mid price to 92
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 39     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 10636        | 1000           | 266           | 100       | 92        | 92               | 93               | 91             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 92    | 39   | party6 | true   |

    # Move mid price to 91
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 41     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 12276        | 1000           | 307           | 100       | 91        | 91               | 92               | 90             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 91    | 41   | party6 | true   |

    # Move mid price to 90
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 41     | 89    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13915        | 1000           | 348           | 100       | 90        | 90               | 91               | 89             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm2-id | 90    | 41   | party6 | true   |

    # Make sure the volumes match, PnL is expected to be different
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party3   | 1      | -10            | 0            |        |
      | party4   | -1     | 10             | 0            |        |
      | party6   | -347   | 1354           | 0            |        |
      | vamm2-id | 347    | -1354          | 0            | true   |
      | vamm1-id | 347    | 0              | 0            | true   |

  @VAMM
  Scenario: 0090-VAMM-030: The volume quoted to move from price 110 to 90 is the same as the volume to move from 100 to 110 + 100 to 90.
    # start out by moving mid prices to 90 and 110 respectively, these are the volumes required to move the price accordingly
    # We don't need to do this as part of this test, but it serves to show where we get the volumes from
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 347    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
      | party6 | ETH/MAR23 | buy  | 74     | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2999         | 1000           | 75            | 100       | 110       | 110              | 111              | 109            |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13915        | 1000           | 348           | 100       | 90        | 90               | 91               | 89             |

    # Now to move from 110 down to 90, the volume ought to be 421 (=347+74)
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 421    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13915        | 1000           | 348           | 100       | 90        | 90               | 91               | 89             |
