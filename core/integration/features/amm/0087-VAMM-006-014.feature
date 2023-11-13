Feature: Ensure the vAMM positions follow the market correctly. AC's 12 and 14 are currently covered in the 012 file with a simpler book. should be added here as more complex scenarios.

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
      | lp1    | ETH/MAR22 | buy  | 20     | 75    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party3 | ETH/MAR22 | buy  | 10     | 85    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 10     | 90    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |           |
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
  Scenario: 0087-VAMM-006: If other traders trade to move the market mid price to 140 the vAMM has a short position.
    # some orders at 150 which trade, to clear the book, leaves 1 sell order @150 on the book, and add buy order at 130 for mid price of 140
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 2      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 5      | 150   | 4                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 5      | 130   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 239          | 1000           | 6             | 100       | 140       | 140              |
    # see the trades that make the vAMM go short
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 106   | 1    | vamm1-id | true   |
      | party5 | 110   | 1    | party2   |        |
      | party5 | 128   | 2    | vamm1-id | true   |
      | party5 | 150   | 1    | party4   |        |
    And the network moves ahead "1" blocks
    ## vAMM holds short
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party5   | 5      | 128            | 0            |        |
      | party1   | 1      | 50             | 0            |        |
      | party2   | -2     | -90            | 0            |        |
      | party4   | -1     | 0              | 0            |        |
      | vamm1-id | -3     | -88            | 0            | true   |

  @VAMM
  Scenario: 0087-VAMM-007: If other traders trade to move the market mid price to 90 the vAMM has a long position.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 13     | 90    | 2                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 559          | 1000           | 14            | 100       | 90        | 90               |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 3    | party4 | true   |
      | party1   | 90    | 10   | party4 |        |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 11     | -10            | 0            |        |
      | party2   | -1     | 10             | 0            |        |
      | party4   | -13    | 42             | 0            |        |
      | vamm1-id | 3      | -42            | 0            | true   |

  @VAMM
  Scenario: 0087-VAMM-008: If other traders trade to move the market mid price to 150 the vAMM will post no further sell orders above this price, and the vAMM's position notional value will be equal to 4x its total account balance.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 1      | 149   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 6      | 149   | 4                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 239          | 1000           | 6             | 100       | 150       | 150              |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 106   | 1    | vamm1-id | true   |
      | party5 | 110   | 1    | party2   |        |
      | party5 | 128   | 2    | vamm1-id | true   |
      | party5 | 149   | 1    | party4   |        |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party5   | 5      | 124            | 0            |        |
      | party1   | 1      | 49             | 0            |        |
      | party2   | -2     | -88            | 0            |        |
      | party4   | -1     | 0              | 0            |        |
      | vamm1-id | -3     | -85            | 0            | true   |
    # vAMM receives fees, but loses out in the MTM settlement
    And the following transfers should happen:
      | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 1      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 2      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 85     | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN     | ETH/MAR22 | 277    | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |

    # TODO: vamm does not appear to have any notional. Neither party nor alias work.
    #And the AMM "vamm1-id" has the following taker notional "4000"
    #And the party "vamm1" has the following taker notional "4000"

  @VAMM
  Scenario: 0087-VAMM-009: If other traders trade to move the market mid price to 85 the vAMM will post no further buy orders below this price, and the vAMM's position notional value will be equal to 4x its total account balance.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 26     | 85    | 4                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/MAR22 | sell | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 1079         | 1000           | 27            | 100       | 85        | 85               |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 3    | party4 | true   |
      | party1   | 90    | 10   | party4 |        |
      | vamm1-id | 95    | 3    | party4 | true   |
      | party3   | 85    | 10   | party4 |        |

    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 11     | -65            | 0            |        |
      | party2   | -1     | 15             | 0            |        |
      | party3   | 10     | 0              | 0            |        |
      | party4   | -26    | 137            | 0            |        |
      | vamm1-id | 6      | -87            | 0            | true   |
    # vAMM receives fees, but loses out in the MTM settlement
    And the following transfers should happen:
      | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 2      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 2      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 87     | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN     | ETH/MAR22 | 315    | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |

    # Now make sure we don't trade with vAMM below 85
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 19     | 75    | 2                | TYPE_LIMIT | TIF_GTC |
    # vAMM closes its position, but no more
    Then the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 89    | 4    | party4 | true   |
      | lp1      | 75    | 15   | party4 | false  |
    When the network moves ahead "1" blocks
    # position is zero for vamm1-id
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 11     | -175           | 0            |        |
      | party2   | -1     | 25             | 0            |        |
      | party3   | 10     | -100           | 0            |        |
      | party4   | -45    | 453            | 0            |        |
      | vamm1-id | 0      | 0              | -1006        | true   |
      | lp1      | 15     | 0              | 0            |        |
    # TODO: vamm does not appear to have any notional. Neither party nor alias work.
    #And the AMM "vamm1-id" has the following taker notional "4000"
    #And the party "vamm1" has the following taker notional "4000"

  @VAMM
  Scenario: 0087-VAMM-010: If other traders trade to move the market mid price to 110 and then trade to move the mid price back to 100 the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 2      | 110   | 2                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 1      | 130   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 119          | 1000           | 3             | 100       | 110       | 110              |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 106   | 1    | vamm1-id | true   |
      | party4 | 110   | 1    | party2   |        |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 10             | 0            |        |
      | party2   | -2     | -10            | 0            |        |
      | party4   | 2      | 4              | 0            |        |
      | vamm1-id | -1     | -4             | 0            | true   |
    # now return the price back to 100, vAMM should hold position of 0
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 3      | 110   | 2                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 110        | TRADING_MODE_CONTINUOUS | 175          | 1000           | 4             | 100       | 100       | 100              |
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 120   | 1    | party2 | true   |
      | party4   | 110   | 1    | party2 |        |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 1      | 10             | 0            |        |
      | party2   | -4     | 0              | 0            |        |
      | party4   | 3      | 4              | 0            |        |
      | vamm1-id | 0      | 0              | -14          | true   |

  @VAMM
  Scenario: 0087-VAMM-011: If other traders trade to move the market mid price to 90 and then trade to move the mid price back to 100 the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 13     | 90    | 2                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 559          | 1000           | 14            | 100       | 90        | 90               |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 3    | party4 | true   |
      | party1   | 90    | 10   | party4 |        |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 11     | -10            | 0            |        |
      | party2   | -1     | 10             | 0            |        |
      | party4   | -13    | 42             | 0            |        |
      | vamm1-id | 3      | -42            | 0            | true   |
    # move price back up to 100
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/MAR22 | buy  | 5      | 100   | 2                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 1      | 90    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 90         | TRADING_MODE_CONTINUOUS | 575          | 1000           | 16            | 100       | 100       | 100              |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 96    | 4    | vamm1-id | true   |
      | party5 | 95    | 1    | party4   |        |

    When the network moves ahead "1" blocks
    # vAMM should not hold a position, but apparently it does, vAMM switched sides, this is a know bug with incoming fix
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 11     | 45             | 0            |        |
      | party2   | -1     | 5              | 0            |        |
      | party4   | -14    | -23            | 0            |        |
      | party5   | 5      | -4             | 0            |        |
      | vamm1-id | -1     | 1              | -24          | true   |

  @VAMM
  Scenario: 0087-VAMM-013: If other traders trade to move the market mid price to 90 and then move the mid price back to 100 in several trades of varying size, the vAMM will have a position of 0.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | sell | 13     | 90    | 2                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | sell | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 559          | 1000           | 14            | 100       | 90        | 90               |
    # see the trades that make the vAMM go short
    And the following trades should be executed:
      | buyer    | price | size | seller | is amm |
      | vamm1-id | 104   | 3    | party4 | true   |
      | party1   | 90    | 10   | party4 |        |
    When the network moves ahead "1" blocks
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 11     | -10            | 0            |        |
      | party2   | -1     | 10             | 0            |        |
      | party4   | -13    | 42             | 0            |        |
      | vamm1-id | 3      | -42            | 0            | true   |
    # move price back up to 100 in several trades at different prices
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 1      | 90    | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 1      | 95    | 1                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/MAR22 | buy  | 3      | 100   | 2                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 90         | TRADING_MODE_CONTINUOUS | 539          | 1000           | 15            | 100       | 100       | 100              |
    And the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party5 | 93    | 1    | vamm1-id | true   |
      | party5 | 95    | 1    | party4   |        |
      | party5 | 97    | 2    | vamm1-id | true   |

    When the network moves ahead "1" blocks
    # vAMM has a position of 0
	Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party1   | 11     | 67             | 0            |        |
      | party2   | -1     | 3              | 0            |        |
      | party4   | -14    | -51            | 0            |        |
      | party5   | 4      | 6              | 0            |        |
      | vamm1-id | 0      | 0              | -25          | true   |
