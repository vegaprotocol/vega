Feature: Test vAMM implied commitment is working as expected

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
      | 0.05        | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

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
      | vamm2  | USD   | 1000000 |

    When the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 100000            | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 100000            | 0.02 | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 20     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |

    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |

    Then the network moves ahead "1" blocks
    And the current epoch is "0"

  @VAMM
  Scenario: 0042-LIQF-107: a vAMM which was active on the market throughout the epoch but with an active range which never overlapped with the SLA range is counted with an implied commitment of `0`.

    # make a one sided AMM away from everything
    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | upper bound | proposed fee | error |
      | vamm1 | ETH/MAR22 | 100000 | 0.5      | 120  | 121         | 0.03         |       |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 120  | 121         |
    Then the network moves ahead "1" blocks

    Then the network moves ahead "1" epochs

    # place some orders around 100
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party1 | ETH/MAR22 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party2 | ETH/MAR22 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 10     | 101   | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    And the following transfers should happen:
      | type                                    | from   | to     | from account                   | to account                     | market id | amount | asset |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE    | market | lp1    | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 10     | USD   |
      | TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE    | market | lp2    | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 10     | USD   |

    # the ELS of the two LPs should be 0.5 implying that the AMM should have 0 share
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
    | party  | equity like share  | virtual stake           | average entry valuation |
    | lp1    | 0.5                | 100000.0000000000000000 | 100000                  |
    | lp2    | 0.5                | 100000.0000000000000000 | 200000                  |