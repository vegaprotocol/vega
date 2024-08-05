Feature: vAMM has the same ELS as liquidity provision with the same commitment amount

  Background:
    Given the average block duration is "1"
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 0.5           | 0.6            | 1.0            |
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
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0     |
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
      | lp_1 | lp1   | ETH/MAR22 | 10000             | 0.02  | submission |

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
      
    Then the network moves ahead "1" epochs
    And the current epoch is "1"

  @VAMM
  Scenario: 0090-VAMM-016: A vAMM's virtual ELS should be equal to the ELS of a regular LP with the same committed volume on the book (i.e. if a vAMM has an average volume on each side of the book across the epoch of 10k USDT, their ELS should be equal to that of a regular LP who has a commitment which requires supplying 10k USDT who joined at the same time as them).

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 39           | 10000          | 0             | 100       | 100       | 100              |

    When the parties submit the following liquidity provision:
      # Using 9788 instead of exactly 10,000 makes things easier because getting exactly 10,000 from an AMM pool as virtual stake can be tricky due to complex math.
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_2 | lp2   | ETH/MAR22 | 9887              | 0.03  | submission |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | proposed fee |
      | vamm1 | ETH/MAR22 | 10000  | 0.8      | 100  | 95          | 105         | 1.041          | 1.041          | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 10000  | STATUS_ACTIVE | 100  | 95          | 105         | 1.041          | 1.041          |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 10000  | USD   | true   | TRANSFER_TYPE_AMM_LOW |

    Then the network moves ahead "1" epochs
    And the current epoch is "2"
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party                                                            | equity like share  | virtual stake         | average entry valuation |
      | lp2                                                              | 0.3320682474642305 | 9887.0000000000000000 | 29774                   |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 0.3320682474642305 | 9887.0000000000000000 | 19887                   |  
  
  @VAMM
  Scenario: 0090-VAMM-017: A vAMM's virtual ELS should be equal to the ELS of a regular LP with the same committed volume on the book (i.e. if a vAMM has an average volume on each side of the book across the epoch of 10k USDT, their ELS should be equal to that of a regular LP who has a commitment which requires supplying 10k USDT who joined at the same time as them).

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 39           | 10000          | 0             | 100       | 100       | 100              |

    When the parties submit the following liquidity provision:
      # Using 10,093 instead of exactly 10,000 makes things easier because getting exactly 10,000 from an AMM pool as virtual stake can be tricky due to complex math.
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_2 | lp2   | ETH/MAR22 | 9887              | 0.03  | submission |

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif   |
      | lp2   | ETH/MAR22 | buy  | 1000   | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp2   | ETH/MAR22 | sell | 1000   | 100   | 0                | TYPE_LIMIT | TIF_GTC |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | proposed fee |
      | vamm1 | ETH/MAR22 | 10000  | 0.05     | 100  | 95          | 105         | 1.041          | 1.041          | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
      | vamm1 | ETH/MAR22 | 10000  | STATUS_ACTIVE | 100  | 95          | 105         | 1.041          | 1.041          |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
    And the following transfers should happen:
      | from  | from account         | to       | to account           | market id | amount | asset | is amm | type                  |
      | vamm1 | ACCOUNT_TYPE_GENERAL | vamm1-id | ACCOUNT_TYPE_GENERAL |           | 10000  | USD   | true   | TRANSFER_TYPE_AMM_LOW |

    Then the network moves ahead "1" epochs
    And the current epoch is "2"
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party                                                            | equity like share  | virtual stake         | average entry valuation |
      | lp2                                                              | 0.3320682474642305 | 9887.0000000000000000 | 29774                   |
      | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 0.3320682474642305 | 9887.0000000000000000 | 19887                   |

  Then the network moves ahead "2" epochs
  And the current epoch is "4"

  And the liquidity provider fee shares for the market "ETH/MAR22" should be:
    | party                                                            | equity like share  | virtual stake         | average entry valuation |
    | lp2                                                              | 0.3320682474642305 | 9887.0000000000000000 | 29774                   |
    | 137112507e25d3845a56c47db15d8ced0f28daa8498a0fd52648969c4b296aba | 0.3320682474642305 | 9887.0000000000000000 | 19887                   |
