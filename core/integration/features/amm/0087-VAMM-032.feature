Feature: vAMM behaviour when a market settles with distressed AMM.

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

    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethMar22Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethMar22Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethMar22Oracle" is given in "0" decimal places

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | ethMar22Oracle     | 1e0                    | 0                         | SLA-22     |

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
      | vamm1 | ETH/MAR22 | 1000   | 0.1      | 100  | 85          | 150         | 0.25               | 0.25               | 0.01         |
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
  Scenario: 0087-VAMM-032: When an AMM is active on a market at time of settlement with a position in a well collateralised state, the market can settle successfully and then all funds on the AMM key are transferred back to the main party's account.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/MAR22 | buy  | 5      | 110   | 1                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/MAR22 | buy  | 5      | 125   | 1                | TYPE_LIMIT | TIF_GTC |
    # see the trades that make the vAMM go short
    Then debug trades
    Then the following trades should be executed:
      | buyer  | price | size | seller   | is amm |
      | party4 | 106   | 1    | vamm1-id | true   |
      | party4 | 120   | 1    | vamm1-id | true   |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | mid price | static mid price | supplied stake | target stake |
      | 120        | TRADING_MODE_CONTINUOUS | 142       | 142              | 1000           | 143          |
    And the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl | is amm |
      | party4   | 2      | 14             | 0            |        |
      | vamm1-id | -2     | -14            | 0            | true   |
    And the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower margin ratio | upper margin ratio |
      | vamm1 | ETH/MAR22 | 1000   | STATUS_ACTIVE | 100  | 85          | 150         | 0.25               | 0.25               |

    # No terminate && settle the market
    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    Then the network moves ahead "1" blocks
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 0       |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 484     | 504    | true   |

    # Settlement price is ~9x mark price
    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 1000  |
    Then the market state should be "STATE_SETTLED" for the market "ETH/MAR22"
    And then the network moves ahead "1" blocks

    # verify the that the margin balance is released, and then the correct balance if transferred from the pool account back to the party.
    # We see fees on both trades, a MTM loss transfer, margin being allocated, then the loss transfer from the final settlement
    # and lastly a transfer of 0 to the general account, indicating there were no funds left to transfer.
    And the following transfers should happen:
      | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                                 |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 1      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE      |
      |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 1      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE      |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 14     | USD   | true   | TRANSFER_TYPE_MTM_LOSS               |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN     | ETH/MAR22 | 504    | USD   | true   | TRANSFER_TYPE_MARGIN_LOW             |
      | vamm1-id | ACCOUNT_TYPE_MARGIN     |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 504    | USD   | true   | TRANSFER_TYPE_LOSS                   |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 484    | USD   | true   | TRANSFER_TYPE_LOSS                   |
      | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1    | ACCOUNT_TYPE_GENERAL    |           | 0      | USD   | true   | TRANSFER_TYPE_AMM_SUBACCOUNT_RELEASE |
    And the parties should have the following account balances:
      | party    | asset | market id | general | margin | is amm |
      | vamm1    | USD   |           | 0       |        |        |
      | vamm1-id | USD   | ETH/MAR22 | 0       | 0      | true   |
