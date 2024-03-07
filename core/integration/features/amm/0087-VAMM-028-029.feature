Feature: Ensure no difference between vAMM moving mid price in one trade is no different to moving in steps.

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
      | ETH/MAR23 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    # Setting up the accounts and vAMM submission now is part of the background, because we'll be running scenarios 0087-VAMM-006 through 0087-VAMM-014 on this setup
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
      | vamm1  | USD   | 1000    |
      | vamm2  | USD   | 1000    |

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
      | lp1    | ETH/MAR22 | buy  | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp3    | ETH/MAR23 | buy  | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC | lp3-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party3 | ETH/MAR23 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party4 | ETH/MAR23 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 10     | 150   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |
      | lp3    | ETH/MAR23 | sell | 10     | 150   | 0                | TYPE_LIMIT | TIF_GTC | lp13s     |
    When the network moves ahead "1" blocks
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
      | party | market id | amount | slippage | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 1000   | 0.1      | 100  | 85          | 150         | 0.25               | 0.25               |
      | vamm2 | ETH/MAR23 | 1000   | 0.1      | 100  | 85          | 150         | 0.25               | 0.25               |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 1000   | STATUS_ACTIVE | 100  | 85          | 150         | 0.25               | 0.25               |
      | vamm2 | ETH/MAR23 | 1000   | STATUS_ACTIVE | 100  | 85          | 150         | 0.25               | 0.25               |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR23 | vamm2-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                             |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 1000   | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_LOW |
      | vamm2 | ACCOUNT_TYPE_GENERAL | vamm2-id | ACCOUNT_TYPE_GENERAL |           | 1000   | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_LOW |

  @VAMM3
  Scenario: 0087-VAMM-028: When a user with 1000 USDT creates a vAMM with commitment 1000, base price 100, upper price 150, lower price 85 and leverage ratio at each bound 0.25, the volume quoted to move from price 100 to price 110 in one step is the same as the sum of the volumes to move in 10 steps of 1 e.g. 100 -> 101, 101 -> 102 etc.
    # First move the mid price on ETH/MAR22 to 110 in a single step
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR22 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 3      | 109   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 106   | 1    | vamm1-id | true   |
    And the network moves ahead "1" blocks
    # Check best offer/bid as this scenario matches 0087-VAMM-027: if other traders trade to move the market mid price to 140 quotes with a mid price of 140 (volume quotes above 140 should be sells, volume quotes below 140 should be buys).
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 106        | TRADING_MODE_CONTINUOUS | 110       | 110              | 111              | 109            |
    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party5   | 1      | 0              | 0            |        |
      | vamm1-id | -1     | 0              | 0            | true   |

    # Now move the mid price step by step, we start with bid/ask of 50/150
    # Move to 101, this will not result in any trades, just orders.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR23 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party6 | ETH/MAR23 | sell | 1      | 102   | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 101       | 101              | 102              | 100            |

    # move to 102, this requires a trade:
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 104   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 1      | 102   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 102   | 1    | party6 |        |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 102        | TRADING_MODE_CONTINUOUS | 102       | 102              | 104              | 100            |

    # Move to 103: will require trades, this is where the vAMM opens a position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 106   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 2      | 104   | 2                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 106   | 1    | vamm2-id | true   |
      | party5 | 104   | 1    | party6   | true   |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 104        | TRADING_MODE_CONTINUOUS | 103       | 103              | 106              | 100            |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -1     | 2              | 0            | true   |

    # Move to 104: more trades, the vAMM closes its position, opens a new one at a different price.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | party6 | ETH/MAR23 | sell | 1      | 108   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 2      | 106   | 2                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller   | is amm |
      | vamm2-id | 119   | 1    | party6   | true   |
      | party5   | 106   | 1    | party6   |        |
      | party5   | 106   | 1    | vamm2-id | true   |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 106        | TRADING_MODE_CONTINUOUS | 104       | 104              | 108              | 100            |
    # Realised PnL indicates the AMM has closed its position, then opened a new one
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -1     | 0              | -13          | true   |

    # Move to 105: more trades, the vAMM closes its position, opens a new one at a different price.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 1      | 108   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 108   | 1    | party6 |        |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 108        | TRADING_MODE_CONTINUOUS | 105       | 105              | 110              | 100            |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -1     | -2             | -13          | true   |

    # move to 106
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 112   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 110   | 1    | party6 |        |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 110        | TRADING_MODE_CONTINUOUS | 106       | 106              | 112              | 100            |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -1     | -4             | -13          | true   |

    # Move to 107
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 114   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 1      | 112   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 112   | 1    | party6 |        |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 112        | TRADING_MODE_CONTINUOUS | 107       | 107              | 114              | 100            |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -1     | -6             | -13          | true   |

    # move to 108
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 116   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 1      | 114   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 114   | 1    | party6 |        |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 114        | TRADING_MODE_CONTINUOUS | 108       | 108              | 116              | 100            |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -1     | -8             | -13          | true   |

    # Move to 109, like when we moved to 104, the AMM will close its position and open a new one again.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party6 | ETH/MAR23 | sell | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | p6-100    |
      | party6 | ETH/MAR23 | sell | 1      | 118   | 0                | TYPE_LIMIT | TIF_GTC | p6-118    |
      | party5 | ETH/MAR23 | buy  | 3      | 116   | 2                | TYPE_LIMIT | TIF_GTC | p5-116    |
    Then the following trades should be executed:
      | buyer    | price | size | seller   | is amm |
      | vamm2-id | 120   | 1    | party6   | true   |
      | party5   | 113   | 2    | vamm2-id | true   |
      | party5   | 116   | 1    | party6   |        |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 116        | TRADING_MODE_CONTINUOUS | 109       | 109              | 118              | 100            |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -2     | -6             | -27          | true   |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party6 | ETH/MAR23 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR23 | buy  | 1      | 118   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 114   | 1    | party6 |        |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 118        | TRADING_MODE_CONTINUOUS | 110       | 110              | 120              | 100            |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | vamm2-id | -2     | -10            | -27          | true   |
