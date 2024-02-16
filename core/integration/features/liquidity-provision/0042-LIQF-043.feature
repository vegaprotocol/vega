Feature: Test LP fee distribution in LP mechanics in 0042

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 1.00             | 20s         | 1              |  
      
    And the following network parameters are set:
      | name                                        | value |
      | market.value.windowLength                   | 60s   |
      | network.markPriceUpdateMaximumFrequency     | 0s    |
      | limits.markets.maxPeggedOrders              | 6     |
      | market.auction.minimumDuration              | 1     |
      | market.fee.factors.infrastructureFee        | 0.001 |
      | market.fee.factors.makerFee                 | 0.004 |
      | spam.protection.max.stopOrdersPerMarket     | 5     |
      | market.liquidity.equityLikeShareFeeFraction | 1     |
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
      | 3600    | 0.99        | 3                 |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.1   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.25  |

    Given the average block duration is "1"
  @Now
  Scenario: 001: lp1 and lp2 on the market ETH/MAR22, 0042-LIQF-043
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | lp3    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 6000              | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 4000              | 0.015 | submission |

    When the network moves ahead "4" blocks
    And the current epoch is "0"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | buy  | 10     | 950   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | sell | 10     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 10000          | 1             |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 1 x 1 x 3.5569036 =3556

    When the network moves ahead "1" epochs
    And the supplied stake should be "9600" for the market "ETH/MAR22"
    And the insurance pool balance should be "400" for the market "ETH/MAR22"
    And the current epoch is "1"
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 56022  | 37978   | 6000 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3600 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the insurance pool balance should be "760" for the market "ETH/MAR22"
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 56022  | 38007   | 6000 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3240 |

    #AC 0042-LIQF-043: The net inflow and outflow into and out of the market's aggregate LP fee account should be zero as a result of penalty collection and bonus distribution.
    #lp1 got lp fee 18 and also SLA performance bonus 11, so general account increased from 37978 to 38007
    #lp2 did not place any orders, so all the lp fee for lp1 goes back to ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION (which was distributed to lp1)

    Then the following transfers should happen:
      | from   | to     | from account                                   | to account                                     | market id | amount | asset |
      | market | lp1    | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 18     | USD   |
      | market | lp2    | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 11     | USD   |
      | lp1    | lp1    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/MAR22 | 18     | USD   |
      | lp2    | market | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/MAR22 | 11     | USD   |
      | market | lp1    | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/MAR22 | 11     | USD   |

  @Now
  Scenario: 001b: lp1 and lp2 on the market ETH/MAR22, 0042-LIQF-043 with fee share factor set to 0.5
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | lp3    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |
    And the following network parameters are set:
      | name                                        | value |
      | market.liquidity.equityLikeShareFeeFraction | 0.5   |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 6000              | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 4000              | 0.015 | submission |

    When the network moves ahead "4" blocks
    And the current epoch is "0"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | buy  | 10     | 950   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | sell | 10     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 10000          | 1             |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 1 x 1 x 3.5569036 =3556

    When the network moves ahead "1" epochs
    And the supplied stake should be "9600" for the market "ETH/MAR22"
    And the insurance pool balance should be "400" for the market "ETH/MAR22"
    And the current epoch is "1"
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 56022  | 37978   | 6000 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3600 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the insurance pool balance should be "760" for the market "ETH/MAR22"
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 56022  | 38007   | 6000 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3240 |

    #AC 0042-LIQF-043: The net inflow and outflow into and out of the market's aggregate LP fee account should be zero as a result of penalty collection and bonus distribution.
    #lp1 got lp fee 18 and also SLA performance bonus 11, so general account increased from 37978 to 38007
    #lp2 did not place any orders, so all the lp fee for lp1 goes back to ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION (which was distributed to lp1)

    Then the following transfers should happen:
      | from   | to     | from account                                   | to account                                     | market id | amount | asset |
      | market | lp1    | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 16     | USD   |
      | market | lp2    | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/MAR22 | 13     | USD   |
      | lp1    | lp1    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/MAR22 | 16     | USD   |
      | lp2    | market | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/MAR22 | 13     | USD   |
      | market | lp1    | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/MAR22 | 13     | USD   |

