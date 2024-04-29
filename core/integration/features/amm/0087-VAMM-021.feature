Feature: Test vAMM cancellation by reduce-only from short.

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
      | party | market id | amount | slippage | base | lower bound | upper bound | lower margin ratio | upper margin ratio | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.1      | 100  | 85          | 150         | 0.25               | 0.25               | 0.01         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 85          | 150         | 0.25               | 0.25               |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                             |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_LOW |


  @VAMM
  Scenario: 0087-VAMM-021: If a vAMM is cancelled and set in Reduce-Only mode when it is currently short, then it creates no further sell orders even if the current price is below the configured upper price. When one of it's buy orders is executed it still does not produce sell orders, and correctly quotes buy orders from a lower price. When the position reaches 0 the vAMM is closed and all funds are released to the user after the next mark to market.
    # based on 0087-VAMM-008: vAMM creates a position, has some general balance left in general and margin accounts.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/MAR22 | buy  | 500    | 155   | 1                | TYPE_LIMIT | TIF_GTC | p4-first  |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 122   | 317  | vamm1-id | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |

    # trying to trade again causes no trades because the AMM has no more volume
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/MAR22 | buy  | 500    | 150   | 0                | TYPE_LIMIT | TIF_GTC | p4-second |

    # the AMM's mid price has moved to 150, but it has no volume +150 so that best offer comes from the orderbook of 160
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 317    | 0              | 0            |        |
      | vamm1-id | -317   | 0              | 0            | true   |
    # Notional value therefore is 317 * 122
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 122        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |
    
    # vAMM receives fees, but loses out in the MTM settlement
    And the following transfers should happen:
       | from     | from account            | to       | to account           | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 155    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN  | ETH/MAR22 | 81210  | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 18945   | 81210  | true   |

    # Immediate cancellation: return vAMM general balance back to the party, margin is confiscated.
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm1 | ETH/MAR22 | METHOD_REDUCE_ONLY |
    Then the AMM pool status should be:
      | party | market id | amount | status             | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_REDUCE_ONLY | 100  | 85          | 150         | 0.25               | 0.25               |
    # Cancel the remaining order from the start of the test
    When the parties cancel the following orders:
      | party  | reference |
      | party4 | p4-first  |
      | party4 | p4-second |
    # Ensure the vAMM cancellation works as expected: a short position shot not increase.
    # Place buy orders at mid price, current mark price, and best bid.
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 10     | 122   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 10     | 149   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 10     | 154   | 0                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 122        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |

    # Now bring in another party that will trade with the buy orders we've just placed, and reduce the exposure of the vAMM
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 350    | 122   | 4                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | party4   | 154   | 10   | party5 |        |
      | party4   | 149   | 10   | party5 |        |
      | party4   | 122   | 10   | party5 |        |
      | vamm1-id | 167   | 149  | party5 | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 347    | -590           | 0            |        |
      | party5   | -179   | 7295           | 0            |        |
      | vamm1-id | -168   | 0              | -6705        | true   |
    And the following transfers should happen:
       | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 100    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 6705   | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 31466  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH       |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 50511   | 43039  | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 122        | TRADING_MODE_CONTINUOUS | 123       | 123              | 122              | 125            |

    # Cool, now close the position of vamm completely
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 40     | 115   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 129   | 40   | party5 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 347    | 1839           | 0            |        |
      | party5   | -219   | 6042           | 0            |        |
      | vamm1-id | -128   | -896           | -6985        | true   |
    And the following transfers should happen:
       | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 21     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 1176   | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 7191   | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH       |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 57723   | 34672   | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 129        | TRADING_MODE_CONTINUOUS | 121       | 121              | 122              | 120            |

    # OK, zero-out the vAMM
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 129    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 131   | 128  | party5 | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 347    | 2533           | 0            |        |
      | party5   | -347   | 5604           | 0            |        |
      | vamm1-id | 0      | 0              | -8137        | true   |
    And the AMM pool status should be:
      | party | market id | amount | status           | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_CANCELLED | 100  | 85          | 150         | 0.25               | 0.25               |
    And the following transfers should happen:
      | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                                 |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 68     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE      |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 256    | USD   | true   | TRANSFER_TYPE_MTM_LOSS               |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 34416  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH            |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1    | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 92207  | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_RELEASE |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 992207  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 0       | 0      | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 131        | TRADING_MODE_CONTINUOUS | 70        | 70               | 100              | 40             |

  @VAMM
  Scenario: 0087-VAMM-021: Same as the test above, only this time, the final order that closes the vAMM position is bigger than the remaining volume, so we check if the vAMM is cancelled instead of going long.
    # based on 0087-VAMM-008: vAMM creates a position, has some general balance left in general and margin accounts.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/MAR22 | buy  | 500    | 155   | 1                | TYPE_LIMIT | TIF_GTC | p4-first  |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 122   | 317  | vamm1-id | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |

    # trying to trade again causes no trades because the AMM has no more volume
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/MAR22 | buy  | 500    | 150   | 0                | TYPE_LIMIT | TIF_GTC | p4-second |

    # the AMM's mid price has moved to 150, but it has no volume +150 so that best offer comes from the orderbook of 160
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 317    | 0              | 0            |        |
      | vamm1-id | -317   | 0              | 0            | true   |
    # Notional value therefore is 317 * 122
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 122        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |
    
    # vAMM receives fees, but loses out in the MTM settlement
    And the following transfers should happen:
       | from     | from account            | to       | to account           | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 155    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN  | ETH/MAR22 | 81210  | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 18945   | 81210  | true   |

    # Immediate cancellation: return vAMM general balance back to the party, margin is confiscated.
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm1 | ETH/MAR22 | METHOD_REDUCE_ONLY |
    Then the AMM pool status should be:
      | party | market id | amount | status             | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_REDUCE_ONLY | 100  | 85          | 150         | 0.25               | 0.25               |
    # Cancel the remaining order from the start of the test
    When the parties cancel the following orders:
      | party  | reference |
      | party4 | p4-first  |
      | party4 | p4-second |
    # Ensure the vAMM cancellation works as expected: a short position shot not increase.
    # Place buy orders at mid price, current mark price, and best bid.
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 10     | 122   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 10     | 149   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 10     | 154   | 0                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 122        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |

    # Now bring in another party that will trade with the buy orders we've just placed, and reduce the exposure of the vAMM
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 350    | 122   | 4                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | party4   | 154   | 10   | party5 |        |
      | party4   | 149   | 10   | party5 |        |
      | party4   | 122   | 10   | party5 |        |
      | vamm1-id | 167   | 149  | party5 | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 347    | -590           | 0            |        |
      | party5   | -179   | 7295           | 0            |        |
      | vamm1-id | -168   | 0              | -6705        | true   |
    And the following transfers should happen:
       | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 100    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 6705   | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 31466  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH       |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 50511   | 43039  | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 122        | TRADING_MODE_CONTINUOUS | 123       | 123              | 122              | 125            |

    # Cool, now close the position of vamm completely
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 40     | 115   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 129   | 40   | party5 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 347    | 1839           | 0            |        |
      | party5   | -219   | 6042           | 0            |        |
      | vamm1-id | -128   | -896           | -6985        | true   |
    And the following transfers should happen:
       | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 21     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 1176   | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 7191   | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH       |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 57723   | 34672  | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 129        | TRADING_MODE_CONTINUOUS | 121       | 121              | 122              | 120            |

    # OK, zero-out the vAMM
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 130    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 131   | 128  | party5 | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 347    | 2533           | 0            |        |
      | party5   | -347   | 5604           | 0            |        |
      | vamm1-id | 0      | 0              | -8137        | true   |
    And the AMM pool status should be:
      | party | market id | amount | status           | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_CANCELLED | 100  | 85          | 150         | 0.25               | 0.25               |
    And the following transfers should happen:
      | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                                 |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 68     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE      |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 256    | USD   | true   | TRANSFER_TYPE_MTM_LOSS               |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 34416  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH            |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1    | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 92207  | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_RELEASE |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 992207  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 0       | 0      | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 131        | TRADING_MODE_CONTINUOUS | 70        | 70               | 100              | 40             |
