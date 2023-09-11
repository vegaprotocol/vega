Feature: Test LP mechanics when there are multiple liquidity providers, and LPs try to amend liquidity commitment;

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the following network parameters are set:
      | name                                          | value |
      | market.value.windowLength                     | 60s   |
      | market.stake.target.timeWindow                | 20s   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 1     |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | limits.markets.maxPeggedOrders                | 6     |
      | market.auction.minimumDuration                | 1     |
      | market.fee.factors.infrastructureFee          | 0.001 |
      | market.fee.factors.makerFee                   | 0.004 |
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
      | 0.5 | 0.6 | 1 | 1.0 |
    And the liquidity sla params named "SLA-23":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0           | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22     |
      | ETH/MAR23 | USD        | USD   | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-23     |

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.1 |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.25  |

    Given the average block duration is "1"
  @Now
  Scenario: 001: lp1 and lp2 on the market ETH/MAR22, 0044-LIME-063, 0044-LIME-079, 0044-LIME-081, 0044-LIME-101
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | lp3 | USD | 100000 |
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
      | lp2    | ETH/MAR22 | buy  | 10     | 970   | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
      | lp2    | ETH/MAR22 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | lp2-s     |
      | lp1    | ETH/MAR22 | sell | 10     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 10000          | 1             |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 1 x 1 x 3.5569036 =3556

    Then the parties cancel the following orders:
      | party | reference |
      | lp1   | lp1-b     |
      | lp2   | lp2-b     |
      | lp2   | lp2-s     |
      | lp1   | lp1-s     |
    #GFA orders got rejected during continous mode
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | error                                        |
      | lp1   | ETH/MAR22 | buy  | 10     | 950   | 0                | TYPE_LIMIT | TIF_GFA | gfa order received during continuous trading |
      | lp2   | ETH/MAR22 | buy  | 10     | 970   | 0                | TYPE_LIMIT | TIF_GFA | gfa order received during continuous trading |
      | lp2   | ETH/MAR22 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GFA | gfa order received during continuous trading |
      | lp1   | ETH/MAR22 | sell | 10     | 1050  | 0                | TYPE_LIMIT | TIF_GFA | gfa order received during continuous trading |

    When the network moves ahead "4" blocks
    And the current epoch is "0"

    #AC 0044-LIME-063: When the LP increases its commitment and the increment is higher than its general account balance, the increments are rejected
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   | error                             |
      | lp_1 | lp1   | ETH/MAR22 | 600000            | 0.02 | amendment | commitment submission not allowed |

    #AC 0044-LIME-101: During continuous trading an LP can submit a transaction to decrease commitment but it will only happen at the end of current epoch.
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 3000              | 0.02 | amendment |
    And the supplied stake should be "10000" for the market "ETH/MAR22"
    When the network moves ahead "4" blocks
    And the supplied stake should be "6000" for the market "ETH/MAR22"
    And the current epoch is "1"

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 0      | 97000   | 2400 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3600 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" blocks
    And the current epoch is "1"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 3600    | 973       | 1027      | 10670        | 6000           | 3             |
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 0      | 97000   | 2400 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3600 |
    #AC 0044-LIME-079: GFA orders during auction from LP count towards LPs liquidity commitment, we check the bond account+general account from the of auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR22 | buy  | 10     | 950   | 0                | TYPE_LIMIT | TIF_GFA | lp1-GFA-b |
      | lp2   | ETH/MAR22 | buy  | 10     | 970   | 0                | TYPE_LIMIT | TIF_GFA | lp2-GFA-b |
      | lp2   | ETH/MAR22 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GFA | lp2-GFA-s |
      | lp1   | ETH/MAR22 | sell | 10     | 1050  | 0                | TYPE_LIMIT | TIF_GFA | lp1-GFA-s |
    Then the network moves ahead "1" epochs
    And the current epoch is "2"

    #margin accont for lp1 and lp2 increased after placing GFA orders during liquidity auction
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 56022  | 40978   | 2400 |
      | lp2   | USD   | ETH/MAR22 | 54421  | 41579   | 3600 |

    #lp3 mades a new LP commitment and get the market out of liquidity auction
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_3 | lp3   | ETH/MAR22 | 5000              | 0.02 | submission |
    When the network moves ahead "5" blocks
    And the current epoch is "2"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 10670        | 11000          | 3             |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 0      | 97000   | 2400 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3600 |
    And the insurance pool balance should be "1030" for the market "ETH/MAR22"

    Then the network moves ahead "1" epochs
    And the current epoch is "3"
    #AC 0044-LIME-081: GFA orders during continuous trading mode from LP do not count towards the LP's liquidity commitment
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 0      | 97000   | 2194 |
      | lp2   | USD   | ETH/MAR22 | 0      | 96000   | 3290 |

    Then the following transfers should happen:
      | from | to     | from account      | to account             | market id | amount | asset |
      | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_INSURANCE | ETH/MAR22 | 206    | USD   |
      | lp2  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_INSURANCE | ETH/MAR22 | 310    | USD   |
    And the insurance pool balance should be "1546" for the market "ETH/MAR22"




