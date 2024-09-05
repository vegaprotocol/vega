Feature: Test vAMM submission works as expected (invalid submission)

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
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.95        | 3                 |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

  @VAMM
  Scenario: 0090-VAMM-004: When market.amm.minCommitmentQuantum is 1, mid price of the market 100, a user with 100 USDT is unable to create a vAMM with commitment 1000, and any other combination of settings.
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | lp3    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |
      | vamm1  | USD   | 100000 |

    When the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 600               | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 400               | 0.015 | submission |
    Then the network moves ahead "4" blocks
    And the current epoch is "0"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/MAR22 | buy  | 10     | 85    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 10     | 90    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party3 | ETH/MAR22 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | buy  | 10     | 95    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | sell | 10     | 105   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | ref price | mid price | static mid price |
      | 100        | TRADING_MODE_CONTINUOUS | 3600    | 94        | 106       | 39           | 1000           | 1             | 100       | 100       | 100              |
    # Try all submissions from AC's 0090-VAMM-001 through 0090-VAMM-003, add some more for good measure
    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | error                                    | proposed fee |
      | vamm1 | ETH/MAR22 | 200000 | 0.1      | 100  | 90          | 110         | 4              | 4              | not enough collateral in general account | 0.01         |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | lower leverage | error                                    | proposed fee |
      | vamm1 | ETH/MAR22 | 200000 | 0.1      | 90   | 85          | 0.25           | not enough collateral in general account | 0.01         |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | upper bound | upper leverage | error                                    | proposed fee |
      | vamm1 | ETH/MAR22 | 200000 | 0.1      | 110  | 150         | 0.25           | not enough collateral in general account | 0.01         |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | lower leverage | error                                    | proposed fee |
      | vamm1 | ETH/MAR22 | 200000 | 0.01     | 110  | 99          | 0.1            | not enough collateral in general account | 0.01         |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | upper bound | upper leverage | error                                    | proposed fee |
      | vamm1 | ETH/MAR22 | 200000 | 0.01     | 90   | 101         | 0.02           | not enough collateral in general account | 0.01         |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage | error                                    | proposed fee |
      | vamm1 | ETH/MAR22 | 200000 | 0.001    | 101  | 95          | 105         | 0.01           | 0.01           | not enough collateral in general account | 0.01         |
