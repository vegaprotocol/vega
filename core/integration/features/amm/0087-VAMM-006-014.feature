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
      | party | market id | amount | slippage | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 100000 | 0.1      | 100  | 85          | 150         | 0.25               | 0.25               |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 85          | 150         | 0.25               | 0.25               |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                             |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_LOW |

  @VAMM3
  Scenario: 0087-VAMM-006: If other traders trade to move the market mid price to 140 the vAMM has a short position.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 265    | 141   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 118   | 265  | vamm1-id | true   |
    And the network moves ahead "1" blocks
    # Check best offer/bid as this scenario matches 0087-VAMM-027: if other traders trade to move the market mid price to 140 quotes with a mid price of 140 (volume quotes above 140 should be sells, volume quotes below 140 should be buys).
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 118        | TRADING_MODE_CONTINUOUS | 140       | 140              | 141              | 139            |
    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 265    | 0              | 0            |        |
      | vamm1-id | -265   | 0              | 0            | true   |

  @VAMM
  Scenario: 0087-VAMM-007: If other traders trade to move the market mid price to 90 the vAMM has a long position.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 350    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go long
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 105   | 350  | party4 | true   |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price |
      | 105        | TRADING_MODE_CONTINUOUS | 90        | 90               | # TODO why isn't this 90?
	  Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -350   | 0              | 0            |        |
      | vamm1-id | 350    | 0              | 0            | true   |

  @VAMM
  Scenario: 0087-VAMM-008: If other traders trade to move the market mid price to 150 the vAMM will post no further sell orders above this price, and the vAMM's position notional value will be equal to 4x its total account balance.
    #When the network moves ahead "1" epochs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 500    | 155   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 122   | 317  | vamm1-id | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 154       | 154              | 160              | 149            |

    # trying to trade again causes no trades because the AMM has no more volume
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 500    | 150     | 0                | TYPE_LIMIT | TIF_GTC |

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
       | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
       |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 155    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
       | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN     | ETH/MAR22 | 81210  | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 1      | 160   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party5 | 160   | 1    | lp1    | false  |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 317    | 12046          | 0            |        |
      | party5   | 1      | 0              | 0            |        |
      | lp1      | -1     | 0              | 0            |        |
      | vamm1-id | -317   | -12046         | 0            | true   |
    # Notional value therefore is 317 * 122

  @VAMM
  Scenario: 0087-VAMM-009: If other traders trade to move the market mid price to 85 the vAMM will post no further buy orders below this price, and the vAMM's position notional value will be equal to 4x its total account balance.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 500    | 80    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 20033        | 1000           | 501           | 100       | 86        | 86               | 87               | 85             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 107   | 500  | party4 | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 7              | 0            |        |
      | party2   | -1     | -7             | 0            |        |
      | party4   | -500   | 0              | 0            |        |
      | vamm1-id | 500    | 0              | 0            | true   |
    # vAMM receives fees, but loses out in the MTM settlement
    And the following transfers should happen:
      | from     | from account            | to       | to account           | market id | amount | asset | is amm | type                            |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 214    | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN  | ETH/MAR22 | 100214 | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |

    # Now make sure we don't trade with vAMM below 85
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 10     | 75    | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | buy  | 10     | 75    | 1                | TYPE_LIMIT | TIF_GTC |

    # vAMM closes its position, but no more
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party3 | 75    | 10   | party4 | false  |
    When the network moves ahead "1" blocks
    # position is zero for vamm1-id
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -25            | 0            |        |
      | party2   | -1     | 25             | 0            |        |
      | party3   | 10     | 0              | 0            |        |
      | party4   | -510   | 16000          | 0            |        |
      | vamm1-id | 500    | -16000         | 0            | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 75         | TRADING_MODE_CONTINUOUS | 15325        | 1000           | 511           | 100       | 86        | 86               | 87               | 85             |
    # TODO: vamm does not appear to have any notional. Neither party nor alias work.
    #And the AMM "vamm1-id" has the following taker notional "4000"
    #And the party "vamm1" has the following taker notional "4000"

  @VAMM
  Scenario: 0087-VAMM-010: If other traders trade to move the market mid price to 110 and then trade to move the mid price back to 100 the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 81     | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 3279         | 1000           | 82            | 100       | 111       | 111              | 112              | 110            |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 104   | 81   | vamm1-id | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party4   | 81     | 0              | 0            |        |
      | vamm1-id | -81    | 0              | 0            | true   |
    # now return the price back to 100, vAMM should hold position of 0
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 81     | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 104        | TRADING_MODE_CONTINUOUS | 3410         | 1000           | 82            | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 115   | 81   | party3 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 15             | 0            |        |
      | party2   | -1     | -15            | 0            |        |
      | party3   | -81    | 0              | 0            |        |
      | party4   | 81     | 891            | 0            |        |
      | vamm1-id | 0      | 0              | -891         | true   |

  @VAMM
  Scenario: 0087-VAMM-011: If other traders trade to move the market mid price to 90 and then trade to move the mid price back to 100 the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 340    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13635        | 1000           | 341           | 100       | 90        | 90               | 91               | 89             |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 340  | party3 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party3   | -340   | 0              | 0            |        |
      | vamm1-id | 340    | 0              | 0            | true   |
    # move price back up to 100
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 340    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 104        | TRADING_MODE_CONTINUOUS | 14181        | 1000           | 341           | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 95    | 340  | vamm1-id | true   |

    When the network moves ahead "1" blocks
    # vAMM should not hold a position, but apparently it does, vAMM switched sides, this is a know bug with incoming fix
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -5             | 0            |        |
      | party2   | -1     | 5              | 0            |        |
      | party3   | -340   | 3060           | 0            |        |
      | party4   | 340    | 0              | 0            |        |
      | vamm1-id | 0      | 0              | -3060        | true   |

  @VAMM
  Scenario: 0087-VAMM-012: If other traders trade to move the market mid price to 90 and then in one trade move the mid price to 110 then trade to move the mid price back to 100 the vAMM will have a position of 0
    # Move mid price to 90
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 340    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13635        | 1000           | 341           | 100       | 90        | 90               | 91               | 89             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 340  | party3 | true   |
    # Check vAMM position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party3   | -340   | 0              | 0            |        |
      | vamm1-id | 340    | 0              | 0            | true   |

    # In a single trade, move the mid privce to 110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 420    | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 104        | TRADING_MODE_CONTINUOUS | 17508        | 1000           | 421           | 100       | 110       | 110              | 111              | 109            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 96    | 420  | vamm1-id | true   |
    # Check the resulting position, vAMM switched from long to short
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -4             | 0            |        |
      | party2   | -1     | 4              | 0            |        |
      | party3   | -340   | 2720           | 0            |        |
      | party4   | 420    | 0              | 0            |        |
      | vamm1-id | -80    | 0              | -2720        | true   |

    # Now return the mid price back to 100
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 80     | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 96         | TRADING_MODE_CONTINUOUS | 16161        | 1000           | 421           | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 115   | 80   | party5 | true   |
    # Check the resulting position, vAMM should hold 0
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 15             | 0            |        |
      | party2   | -1     | -15            | 0            |        |
      | party3   | -340   | -3740          | 0            |        |
      | party4   | 420    | 7980           | 0            |        |
      | party5   | -80    | 0              | 0            |        |
      | vamm1-id | 0      | 0              | -4240        | true   |

  @VAMM
  Scenario: 0087-VAMM-013: If other traders trade to move the market mid price to 90 and then move the mid price back to 100 in several trades of varying size, the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 340    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13635        | 1000           | 341           | 100       | 90        | 90               | 91               | 89             |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 340  | party3 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party3   | -340   | 0              | 0            |        |
      | vamm1-id | 340    | 0              | 0            | true   |
    # move price back up to 100, in several trades of varying sizes
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 99     | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 121    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 120    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 104        | TRADING_MODE_CONTINUOUS | 14181        | 1000           | 341           | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party1 | 91    | 99   | vamm1-id | true   |
      | party4 | 94    | 121  | vamm1-id | true   |
      | party5 | 98    | 120  | vamm1-id | true   |

    When the network moves ahead "1" blocks
    # vAMM should not hold a position, but apparently it does, vAMM switched sides, this is a know bug with incoming fix
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 100    | 691            | 0            |        |
      | party2   | -1     | 2              | 0            |        |
      | party3   | -340   | 2040           | 0            |        |
      | party4   | 121    | 484            | 0            |        |
      | party5   | 120    | 0              | 0            |        |
      | vamm1-id | 0      | 0              | -3217        | true   |

  @VAMM
  Scenario: 0087-VAMM-014: If other traders trade to move the market mid price to 90 and then in one trade move the mid price to 110 then trade to move the mid price to 120 the vAMM will have a larger (more negative) but comparable position to if they had been moved straight from 100 to 120.
    # Move mid price to 90
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 340    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 13635        | 1000           | 341           | 100       | 90        | 90               | 91               | 89             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 340  | party3 | true   |
    # Check vAMM position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party3   | -340   | 0              | 0            |        |
      | vamm1-id | 340    | 0              | 0            | true   |

    # In a single trade, move the mid privce to 110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 420    | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 104        | TRADING_MODE_CONTINUOUS | 17508        | 1000           | 421           | 100       | 110       | 110              | 111              | 109            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 96    | 420  | vamm1-id | true   |
    # Check the resulting position, vAMM switched from long to short
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -4             | 0            |        |
      | party2   | -1     | 4              | 0            |        |
      | party3   | -340   | 2720           | 0            |        |
      | party4   | 420    | 0              | 0            |        |
      | vamm1-id | -80    | 0              | -2720        | true   |

    # Now further increase the mid price, move it up to 120
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 65     | 120   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 96         | TRADING_MODE_CONTINUOUS | 18656        | 1000           | 486           | 100       | 120       | 120              | 121              | 119            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 114   | 65   | vamm1-id | true   |
    # Check the resulting position, vAMM further increased their position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 14             | 0            |        |
      | party2   | -1     | -14            | 0            |        |
      | party3   | -340   | -3400          | 0            |        |
      | party4   | 420    | 7560           | 0            |        |
      | party5   | 65     | 0              | 0            |        |
      | vamm1-id | -145   | -1440          | -2720        | true   |
