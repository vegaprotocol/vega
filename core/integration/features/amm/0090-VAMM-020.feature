Feature: Test vAMM cancellation by reduce-only from long.

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


  @VAMM
  Scenario: 0090-VAMM-020: If a vAMM is cancelled and set in Reduce-Only mode when it is currently long, then It creates no further buy orders even if the current price is above the configured lower price. When one of it's sell orders is executed it still does not produce buy orders, and correctly quotes sell orders from a higher price. When the position reaches 0 the vAMM is closed and all funds are released to the user after the next mark to market.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 350    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go long
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 95    | 350  | party4 | true   |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price |
      | 95         | TRADING_MODE_CONTINUOUS | 90        | 90               |
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -350   | 0              | 0            |        |
      | vamm1-id | 350    | 0              | 0            | true   |
    And the following transfers should happen:
      | from     | from account            | to       | to account           | market id | amount | asset | is amm | type                            |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 133    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN  | ETH/MAR22 | 64462  | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 35671   | 64462  | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 90        | 90               | 91               | 89             |

    # Next: cancel the vAMM with reduce-only
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm1 | ETH/MAR22 | METHOD_REDUCE_ONLY |
    Then the AMM pool status should be:
      | party | market id | amount | status             | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_REDUCE_ONLY | 100  | 85          | 150         | 4              | 4              |
    # Check if the vAMM doesn't place any more buy orders: submit sell orders at previous best bid, ask, and mid prices:
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 10     | 89    | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 10     | 90    | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 10     | 91    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 64        | 64               | 89               | 40             |
    
    # Now start checking if the vAMM still quotes sell orders
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 280    | 110   | 5                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 89    | 10   | party4   |        |
      | party5 | 90    | 10   | party4   |        |
      | party5 | 90    | 19   | vamm1-id | true   |
      | party5 | 91    | 10   | party4   |        |
      | party5 | 90    | 19   | vamm1-id | true   |
      | party5 | 94    | 231  | vamm1-id | true   |

    # check the state of the market, trigger MTM settlement and check balances before closing out the last 100 for the vAMM
    When the network moves ahead "1" blocks
	  Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -380   | 230            | 0            |        |
      | party5   | 280    | 196            | 0            |        |
      | vamm1-id | 100    | -100           | -326         | true   |
    # vAMM is still quoting bid price, though it is in reduce-only mode, and therefore doesn't place those orders.
    # The best bid should be 40 here?
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 94         | TRADING_MODE_CONTINUOUS | 69        | 69               | 98               | 40             |
    # vAMM receives some fees, but pays MTM loss, excess margin is released
    And the following transfers should happen:
      | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 87     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 426    | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 45811  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH       |
    # After receiving fees, and excess margin is correctly released, the balances of the vAMM sub-accounts match the position:
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 81576   | 18225  | true   |

    # Now make sure the vAMM, though clearly having sufficient balance to increase its position, still doesn't place any buy orders (reduce only check 2)
    # Like before, place orders at mid, offer, and bid prices
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/MAR22 | sell | 10     | 96    | 0                | TYPE_LIMIT | TIF_GTC | p4-c1     |
      | party4 | ETH/MAR22 | sell | 10     | 97    | 0                | TYPE_LIMIT | TIF_GTC | p4-c2     |
      | party4 | ETH/MAR22 | sell | 10     | 98    | 0                | TYPE_LIMIT | TIF_GTC | p4-c3     |
    # we've confirmed the vAMM does not reduce its position at all, so cancel these orders to keep things simple
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | p4-c1     |
      | party4 | p4-c2     |
      | party4 | p4-c3     |
    # party5 places a buy order large enough to trade with party4 and reduce the vAMM position down to 0, and no more.
    # we'll do a  v2 of this test where this buy order is "over-sized" to ensure the vAMM doesn't flip from long to short.
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 100    | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 98    | 100  | vamm1-id | true   |
    # Confirm the vAMM is no longer quoting anything
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 94         | TRADING_MODE_CONTINUOUS | 100       | 100              | 160              | 40             |

    # Check the final PnL for the vAMM, check the transfers and balances
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -380   | -1290          | 0            |        |
      | party5   | 380    | 1316           | 0            |        |
      | vamm1-id | 0      | 0              | -26          | true   |
    And the AMM pool status should be:
      | party | market id | amount | status           | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_CANCELLED | 100  | 85          | 150         | 4              | 4              |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 98         | TRADING_MODE_CONTINUOUS | 100       | 100              | 160              | 40             |
    And the following transfers should happen:
       | from     | from account            | to       | to account           | market id | amount | asset | is amm | type                                 |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 40     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE      |
       |          | ACCOUNT_TYPE_SETTLEMENT | vamm1-id | ACCOUNT_TYPE_MARGIN  | ETH/MAR22 | 400    | USD   | true   | TRANSFER_TYPE_MTM_WIN                |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 18625  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH            |
       | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1    | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 100241 | USD   | true   | TRANSFER_TYPE_AMM_RELEASE            |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 1000241 |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 0       | 0      | true   |

  @VAMM
  Scenario: 0090-VAMM-020: Same as the test above, only the final buy order that moves the vAMM position to 0 is more than big enough, and doesn't cause the vAMM to flip position from long to short.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 350    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go long
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 95    | 350  | party4 | true   |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price |
      | 95         | TRADING_MODE_CONTINUOUS | 90        | 90               |
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -350   | 0              | 0            |        |
      | vamm1-id | 350    | 0              | 0            | true   |
    And the following transfers should happen:
      | from     | from account            | to       | to account           | market id | amount | asset | is amm | type                            |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 133    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN  | ETH/MAR22 | 64462  | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 35671   | 64462  | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 90        | 90               | 91               | 89             |

    # Next: cancel the vAMM with reduce-only
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm1 | ETH/MAR22 | METHOD_REDUCE_ONLY |
    Then the AMM pool status should be:
      | party | market id | amount | status             | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_REDUCE_ONLY | 100  | 85          | 150         | 4              | 4              |
    # Check if the vAMM doesn't place any more buy orders: submit sell orders at previous best bid, ask, and mid prices:
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 10     | 89    | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 10     | 90    | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 10     | 91    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 64        | 64               | 89               | 40             |
    And clear trade events
    # Now start checking if the vAMM still quotes sell orders
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 280    | 110   | 5                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 89    | 10   | party4   |        |
      | party5 | 90    | 10   | party4   |        |
      | party5 | 90    | 19   | vamm1-id | true   |
      | party5 | 91    | 10   | party4   |        |
      | party5 | 90    | 19   | vamm1-id | true   |
      | party5 | 94    | 231  | vamm1-id | true   |

    # check the state of the market, trigger MTM settlement and check balances before closing out the last 100 for the vAMM
    When the network moves ahead "1" blocks
	  Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -380   | 230            | 0            |        |
      | party5   | 280    | 196            | 0            |        |
      | vamm1-id | 100    | -100           | -326         | true   |
    # vAMM is still quoting bid price, though it is in reduce-only mode, and therefore doesn't place those orders.
    # The best bid should be 40 here?
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 94         | TRADING_MODE_CONTINUOUS | 69        | 69               | 98               | 40             |
    # vAMM receives some fees, but pays MTM loss, excess margin is released
    And the following transfers should happen:
      | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 87     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 426    | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 45811  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH       |
    # After receiving fees, and excess margin is correctly released, the balances of the vAMM sub-accounts match the position:
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 900000  |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 81576   | 18225  | true   |

    # Now make sure the vAMM, though clearly having sufficient balance to increase its position, still doesn't place any buy orders (reduce only check 2)
    # Like before, place orders at mid, offer, and bid prices
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/MAR22 | sell | 10     | 96    | 0                | TYPE_LIMIT | TIF_GTC | p4-c1     |
      | party4 | ETH/MAR22 | sell | 10     | 97    | 0                | TYPE_LIMIT | TIF_GTC | p4-c2     |
      | party4 | ETH/MAR22 | sell | 10     | 98    | 0                | TYPE_LIMIT | TIF_GTC | p4-c3     |
    # we've confirmed the vAMM does not reduce its position at all, so cancel these orders to keep things simple
    Then the parties cancel the following orders:
      | party  | reference |
      | party4 | p4-c1     |
      | party4 | p4-c2     |
      | party4 | p4-c3     |
    # party5 places a buy order large enough to trade with party4 and reduce the vAMM position down to 0, and more, vAMM does not trade to move from long to short, it stays at 0.
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 150    | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 98    | 100  | vamm1-id | true   |
    # Confirm the vAMM is no longer quoting anything
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 94         | TRADING_MODE_CONTINUOUS | 135       | 135              | 160              | 110            |

    # Check the final PnL for the vAMM, check the transfers and balances
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -380   | -1290          | 0            |        |
      | party5   | 380    | 1316           | 0            |        |
      | vamm1-id | 0      | 0              | -26          | true   |
    And the AMM pool status should be:
      | party | market id | amount | status           | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_CANCELLED | 100  | 85          | 150         | 4              | 4              |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 98         | TRADING_MODE_CONTINUOUS | 135       | 135              | 160              | 110            |
    And the following transfers should happen:
       | from     | from account            | to       | to account           | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 40     | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       |          | ACCOUNT_TYPE_SETTLEMENT | vamm1-id | ACCOUNT_TYPE_MARGIN  | ETH/MAR22 | 400    | USD   | true   | TRANSFER_TYPE_MTM_WIN           |
       | vamm1-id | ACCOUNT_TYPE_MARGIN     | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 18625  | USD   | true   | TRANSFER_TYPE_MARGIN_HIGH       |
       | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1    | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 100241 | USD   | true   | TRANSFER_TYPE_AMM_RELEASE       |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 1000241 |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 0       | 0      | true   |
