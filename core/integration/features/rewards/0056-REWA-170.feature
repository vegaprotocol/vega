Feature: If an AMM sub-key earns rewards, they are transferred into the sub-keys vesting account and locked for the appropriate period before vesting

  Background:
    Given the average block duration is "1"      
    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 60s   |
      | validators.epoch.length                             | 10s   |
      | market.value.windowLength                           | 60s   |
      | network.markPriceUpdateMaximumFrequency             | 2s    |
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
      | rewards.vesting.baseRate                            | 0.1   |
      | rewards.vesting.minimumTransfer                     | 1     |

    And the following assets are registered:
      | id  | decimal places |
      | USD | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params    |
      | ETH/MAR22 | USD        | USD   | default-log-normal-risk-model | default-margin-calculator | 2                | fees-config-1 | default-basic    | default-eth-for-future | 1e0                    | 0                         | default-basic |

  @VAMM
  Scenario: 
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount  |
      | party1                                                           | USD   |  100000 |
      | party2                                                           | USD   |  100000 |
      | party3                                                           | USD   |  100000 |
      | vamm1                                                            | USD   |  100000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | VEGA  | 1000000 |
    And the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                         | asset | amount | start_epoch | end_epoch | factor | metric                              | metric_asset | markets   |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_RECEIVED | USD          | ETH/MAR22 |

    When the network moves ahead "4" blocks
    Then the current epoch is "0"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            |
      | 100        | TRADING_MODE_CONTINUOUS |
    
    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.1      | 100  | 85          | 150         | 0.25           | 0.25           | 0.01         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 85          | 150         | 0.25           | 0.25           |
    And set the following AMM sub account aliases:
      | party | market id | alias     |
      | vamm1 | ETH/MAR22 | vamm1-acc |
    And the following transfers should happen:
      | from  | from account         | to        | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-acc | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |

    When the network moves ahead "11" blocks
    Then the current epoch is "1"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_FOK |
    Then the following trades should be executed:
      | buyer  | price | size | seller                                                           | buyer maker fee |
      | party1 | 100   | 1    | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 1               |

    When the network moves ahead "11" blocks
    Then the current epoch is "2"
    And "137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba" should have vesting account balance of "10000" for asset "VEGA"

    When the network moves ahead "11" blocks
    Then the current epoch is "3"
    And "137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba" should have vesting account balance of "5000" for asset "VEGA"
    And "137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba" should have vested account balance of "5000" for asset "VEGA"

    When the network moves ahead "11" blocks
    Then the current epoch is "4"
    And "137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba" should have vesting account balance of "0" for asset "VEGA"
    And "137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba" should have vested account balance of "10000" for asset "VEGA"