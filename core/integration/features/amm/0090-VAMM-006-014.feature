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
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                             |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |

  @VAMM
  Scenario: 0090-VAMM-006: If other traders trade to move the market mid price to 140 the vAMM has a short position.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 245    | 141   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 118   | 245  | vamm1-id | true   |
    And the network moves ahead "1" blocks
    # Check best offer/bid as this scenario matches 0090-VAMM-027: if other traders trade to move the market mid price to 140 quotes with a mid price of 140 (volume quotes above 140 should be sells, volume quotes below 140 should be buys).
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 118        | TRADING_MODE_CONTINUOUS | 140       | 140              | 141              | 139            |
    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 245    | 0              | 0            |        |
      | vamm1-id | -245   | 0              | 0            | true   |

  @VAMM
  Scenario: 0090-VAMM-007: If other traders trade to move the market mid price to 90 the vAMM has a long position.
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
      | 95         | TRADING_MODE_CONTINUOUS | 90        | 90               | # TODO why isn't this 90?
	  Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | -350   | 0              | 0            |        |
      | vamm1-id | 350    | 0              | 0            | true   |

  @VAMM
  Scenario: 0090-VAMM-008: If other traders trade to move the market mid price to 150 the vAMM will post no further sell orders above this price, and the vAMM's position notional value will be equal to 4x its total account balance.
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


  @VAMM
  Scenario: 0090-VAMM-009: If other traders trade to move the market mid price to 85 the vAMM will post no further buy orders below this price, and the vAMM's position notional value will be equal to 4x its total account balance.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 581    | 80    | 1                | TYPE_LIMIT | TIF_GTC |

    # AMM is at its bound so will have no orders below 85 so best bid will be 40 which is an LP order from the test setup
    # best offer will be 86 which is quoted from the pool
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 63        | 63               | 86               | 40             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 92    | 581  | party4 | true   |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -8             | 0            |        |
      | party2   | -1     | 8              | 0            |        |
      | party4   | -581   | 0              | 0            |        |
      | vamm1-id | 581    | 0              | 0            | true   |
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

    # trade does not happen with the AMM
    Then the following trades should be executed:
      | buyer  | price | size | seller | is amm |
      | party3 | 75    | 10   | party4 | false  |
    When the network moves ahead "1" blocks
	  Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -25            | 0            |        |
      | party2   | -1     | 25             | 0            |        |
      | party3   | 10     | 0              | 0            |        |
      | party4   | -591   | 9877           | 0            |        |
      | vamm1-id | 581    | -9877          | 0            | true   |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | best offer price | best bid price |
      | 75         | TRADING_MODE_CONTINUOUS | 63        | 63               | 86               | 40             |
    # TODO: vamm does not appear to have any notional. Neither party nor alias work.
    #And the AMM "vamm1-id" has the following taker notional "4000"
    #And the party "vamm1" has the following taker notional "4000"

  @VAMM
  Scenario: 0090-VAMM-010: If other traders trade to move the market mid price to 110 and then trade to move the mid price back to 100 the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 74     | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 2999         | 1000           | 75            | 100       | 110       | 110              | 111              | 109            |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 104   | 74   | vamm1-id | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party4   | 74     | 0              | 0            |        |
      | vamm1-id | -74    | 0              | 0            | true   |
    # now return the price back to 100, vAMM should hold position of 0
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 74     | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 104        | TRADING_MODE_CONTINUOUS | 3119         | 1000           | 75            | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 74   | party3 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party3   | -74    | 0              | 0            |        |
      | party4   | 74     | 0              | 0            |        |
      | vamm1-id | 0      | 0              | 0            | true   |

  @VAMM
  Scenario: 0090-VAMM-011: If other traders trade to move the market mid price to 90 and then trade to move the mid price back to 100 the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 371    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 14875        | 1000           | 372           | 100       | 90        | 90               | 91               | 89             |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 94    | 371  | party3 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -6             | 0            |        |
      | party2   | -1     | 6              | 0            |        |
      | party3   | -371   | 0              | 0            |        |
      | vamm1-id | 371    | 0              | 0            | true   |
    # move price back up to 100
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 371    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | ref price | mid price | static mid price | best offer price | best bid price |
      | 94         | TRADING_MODE_CONTINUOUS | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 94    | 371  | vamm1-id | true   |

    When the network moves ahead "1" blocks
    # vAMM should not hold a position, but apparently it does, vAMM switched sides, this is a know bug with incoming fix
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -6             | 0            |        |
      | party2   | -1     | 6              | 0            |        |
      | party3   | -371   | 0              | 0            |        |
      | party4   | 371    | 0              | 0            |        |
      | vamm1-id | 0      | 0              | 0            | true   |

  @VAMM
  Scenario: 0090-VAMM-012: If other traders trade to move the market mid price to 90 and then in one trade move the mid price to 110 then trade to move the mid price back to 100 the vAMM will have a position of 0
    # Move mid price to 90
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 371    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 100       | 90        | 90               | 91               | 89             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 94    | 371  | party3 | true   |
    # Check vAMM position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -6             | 0            |        |
      | party2   | -1     | 6              | 0            |        |
      | party3   | -371   | 0              | 0            |        |
      | vamm1-id | 371    | 0              | 0            | true   |

    # In a single trade, move the mid price to 110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 440    | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | ref price | mid price | static mid price | best offer price | best bid price |
      | 94         | TRADING_MODE_CONTINUOUS | 100       | 110       | 110              | 111              | 109            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 95    | 440  | vamm1-id | true   |
    # Check the resulting position, vAMM switched from long to short
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -5             | 0            |        |
      | party2   | -1     | 5              | 0            |        |
      | party3   | -371   | -371           | 0            |        |
      | party4   | 440    | 0              | 0            |        |
      | vamm1-id | -69    | 0              | 371          | true   |

    # Now return the mid price back to 100
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | sell | 69     | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | ref price | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 69   | party5 | true   |
    # Check the resulting position, vAMM should hold 0
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 4              | 0            |        |
      | party2   | -1     | -4             | 0            |        |
      | party3   | -371   | -3710          | 0            |        |
      | party4   | 440    | 3960           | 0            |        |
      | party5   | -69    | 0              | 0            |        |
      | vamm1-id | 0      | 0              | -250         | true   |

  @VAMM
  Scenario: 0090-VAMM-013: If other traders trade to move the market mid price to 90 and then move the mid price back to 100 in several trades of varying size, the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 350    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 14035        | 1000           | 351           | 100       | 90        | 90               | 91               | 89             |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 95    | 350  | party3 | true   |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -5             | 0            |        |
      | party2   | -1     | 5              | 0            |        |
      | party3   | -350   | 0              | 0            |        |
      | vamm1-id | 350    | 0              | 0            | true   |
    # move price back up to 100, in several trades of varying sizes
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 99     | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 121    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 130    | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | ref price | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 100       | 100       | 100              | 101              | 99             |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party1 | 91    | 99   | vamm1-id | true   |
      | party4 | 94    | 121  | vamm1-id | true   |
      | party5 | 97    | 130  | vamm1-id | true   |

    When the network moves ahead "1" blocks
    # vAMM should not hold a position, but apparently it does, vAMM switched sides, this is a know bug with incoming fix
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 100    | 591            | 0            |        |
      | party2   | -1     | 3              | 0            |        |
      | party3   | -350   | -700           | 0            |        |
      | party4   | 121    | 363            | 0            |        |
      | party5   | 130    | 0              | 0            |        |
      | vamm1-id | 0      | 0              | -257         | true   |

  @VAMM
  Scenario: 0090-VAMM-014: If other traders trade to move the market mid price to 90 and then in one trade move the mid price to 110 then trade to move the mid price to 120 the vAMM will have a larger (more negative) but comparable position to if they had been moved straight from 100 to 120.
    # Move mid price to 90
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/MAR22 | sell | 350    | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price | best offer price | best bid price |
      | 100        | TRADING_MODE_CONTINUOUS | 14035        | 1000           | 351           | 100       | 90        | 90               | 91               | 89             |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 95    | 350  | party3 | true   |
    # Check vAMM position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -5             | 0            |        |
      | party2   | -1     | 5              | 0            |        |
      | party3   | -350   | 0              | 0            |        |
      | vamm1-id | 350    | 0              | 0            | true   |

    # In a single trade, move the mid price to 110
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 420    | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | ref price | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 100       | 110       | 110              | 111              | 109            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 95    | 420  | vamm1-id | true   |
    # Check the resulting position, vAMM switched from long to short
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | -5             | 0            |        |
      | party2   | -1     | 5              | 0            |        |
      | party3   | -350   | 0              | 0            |        |
      | party4   | 420    | 0              | 0            |        |
      | vamm1-id | -70    | 0              | 0            | true   |

    # Now further increase the mid price, move it up to 120
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 65     | 120   | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | ref price | mid price | static mid price | best offer price | best bid price |
      | 95         | TRADING_MODE_CONTINUOUS | 100       | 120       | 120              | 121              | 119            |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 114   | 65   | vamm1-id | true   |
    # Check the resulting position, vAMM further increased their position
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 14             | 0            |        |
      | party2   | -1     | -14            | 0            |        |
      | party3   | -350   | -6650          | 0            |        |
      | party4   | 420    | 7980           | 0            |        |
      | party5   | 65     | 0              | 0            |        |
      | vamm1-id | -135   | -1330          | 0            | true   |
