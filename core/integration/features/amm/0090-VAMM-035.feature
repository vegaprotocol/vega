Feature: 0090-VAMM-035 With two vAMMs existing on the market, and no other orders, both of which have the same fair price, another counterparty placing a large buy order for a given volume, followed by a large sell order for the same volume, results in the vAMMs both taking a position and then returning to 0 position, with a balance increase equal to the maker fees received plus those for the incoming trader crossing the spread.

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

    And the following assets are registered:
      | id  | decimal places |
      | USD | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params    |
      | ETH/MAR22 | USD        | USD   | default-log-normal-risk-model | default-margin-calculator | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | default-basic |

  @VAMM
  Scenario: Double-sided vAMMs
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount  |
      | party1                                                           | USD   |  100000 |
      | party2                                                           | USD   |  100000 |
      | party3                                                           | USD   |  100000 |
      | vamm1                                                            | USD   |  100000 |
      | vamm2                                                            | USD   |  100000 |

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
      | vamm1 | ETH/MAR22 |  50000 | 0.1      | 100  | 85          | 115         | 0.25           | 0.3            | 0.01         |
      | vamm2 | ETH/MAR22 | 100000 | 0.1      | 100  | 85          | 115         | 0.25           | 0.3            | 0.012        |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 |  50000 | STATUS_ACTIVE | 100  | 85          | 115         | 0.25           | 0.3            |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 85          | 115         | 0.25           | 0.3            |
    And set the following AMM sub account aliases:
      | party | market id | alias     |
      | vamm1 | ETH/MAR22 | vamm1-acc |
      | vamm2 | ETH/MAR22 | vamm2-acc |
    And the following transfers should happen:
      | from  | from account         | to        | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-acc | ACCOUNT_TYPE_GENERAL |           |  50000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |
      | vamm2 | ACCOUNT_TYPE_GENERAL | vamm2-acc | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |

    Then the parties should have the following account balances:
      | party                                                            | asset | market id | margin | general | vesting | vested | 
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | USD   | ETH/MAR22 | 0      |  50000  |         |        |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | USD   | ETH/MAR22 | 0      | 100000  |         |        |

    When the network moves ahead "11" blocks
    Then the current epoch is "0"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party1 | ETH/MAR22 | buy  | 100    | 0     | 2                | TYPE_MARKET | TIF_FOK |
    Then the following trades should be executed:
      | buyer  | price | size | seller                                                           |
      | party1 | 101   | 34   | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba |
      | party1 | 101   | 66   | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 |

    When the network moves ahead "2" blocks
    And the parties should have the following profit and loss:
      | party                                                            | volume | unrealised pnl | realised pnl |
      | party1                                                           |  101   |  1             | 0            |
      | party2                                                           | -1     | -1             | 0            |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | -34    |  0             | 0            |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | -66    |  0             | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party2 | ETH/MAR22 | sell | 100    | 0     | 2                | TYPE_MARKET | TIF_FOK |
        Then debug trades
    Then the following trades should be executed:
      | buyer                                                            | price | size | seller |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 101   | 34   | party2 |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | 101   | 66   | party2 |

    When the network moves ahead "2" blocks
    And the parties should have the following profit and loss:
      | party                                                            | volume | unrealised pnl | realised pnl |
      | party1                                                           |  101   |  1             | 0            |
      | party2                                                           | -101   | -1             | 0            |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba |    0   |  0             | 0            |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 |    0   |  0             | 0            |

    Then the parties should have the following account balances:
      | party                                                            | asset | market id | margin | general | vesting | vested | 
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | USD   | ETH/MAR22 | 0      |  50028  |         |        |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | USD   | ETH/MAR22 | 0      | 100054  |         |        |

Scenario: Single-sided vAMMs
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount  |
      | party1                                                           | USD   |  100000 |
      | party2                                                           | USD   |  100000 |
      | party3                                                           | USD   |  100000 |
      | vamm1                                                            | USD   |  100000 |
      | vamm2                                                            | USD   |  100000 |

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
      | vamm1 | ETH/MAR22 |  50000 | 0.1      | 100  |             | 115         |                | 0.3            | 0.01         |
      | vamm2 | ETH/MAR22 | 100000 | 0.1      | 100  |             | 115         |                | 0.3            | 0.012        |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 |  50000 | STATUS_ACTIVE | 100  |             | 115         |                | 0.3            |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  |             | 115         |                | 0.3            |
    And set the following AMM sub account aliases:
      | party | market id | alias     |
      | vamm1 | ETH/MAR22 | vamm1-acc |
      | vamm2 | ETH/MAR22 | vamm2-acc |
    And the following transfers should happen:
      | from  | from account         | to        | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-acc | ACCOUNT_TYPE_GENERAL |           |  50000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |
      | vamm2 | ACCOUNT_TYPE_GENERAL | vamm2-acc | ACCOUNT_TYPE_GENERAL |           | 100000 | USD   | true   | TRANSFER_TYPE_AMM_LOW |

    Then the parties should have the following account balances:
      | party                                                            | asset | market id | margin | general | vesting | vested | 
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | USD   | ETH/MAR22 | 0      |  50000  |         |        |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | USD   | ETH/MAR22 | 0      | 100000  |         |        |

    When the network moves ahead "11" blocks
    Then the current epoch is "0"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party1 | ETH/MAR22 | buy  | 100    | 0     | 2                | TYPE_MARKET | TIF_FOK |
    Then the following trades should be executed:
      | buyer  | price | size | seller                                                           |
      | party1 | 101   | 34   | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba |
      | party1 | 101   | 66   | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 |

    When the network moves ahead "2" blocks
    And the parties should have the following profit and loss:
      | party                                                            | volume | unrealised pnl | realised pnl |
      | party1                                                           |  101   |  1             | 0            |
      | party2                                                           | -1     | -1             | 0            |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | -34    |  0             | 0            |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | -66    |  0             | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party2 | ETH/MAR22 | sell | 100    | 0     | 2                | TYPE_MARKET | TIF_FOK |
        Then debug trades
    Then the following trades should be executed:
      | buyer                                                            | price | size | seller |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 101   | 34   | party2 |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | 101   | 66   | party2 |

    When the network moves ahead "2" blocks
    And the parties should have the following profit and loss:
      | party                                                            | volume | unrealised pnl | realised pnl |
      | party1                                                           |  101   |  1             | 0            |
      | party2                                                           | -101   | -1             | 0            |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba |    0   |  0             | 0            |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 |    0   |  0             | 0            |

    Then the parties should have the following account balances:
      | party                                                            | asset | market id | margin | general | vesting | vested | 
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | USD   | ETH/MAR22 | 0      |  50028  |         |        |
      | 4582953f1f1dd07603befe97994d6414c0ebb53c7d52c29e866bb3e85d7b30b4 | USD   | ETH/MAR22 | 0      | 100054  |         |        |