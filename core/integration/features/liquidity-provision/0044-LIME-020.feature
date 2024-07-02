Feature: Test LP mechanics when there are multiple liquidity providers;

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 1.0              | 20s         | 1              |

    And the following network parameters are set:
      | name                                        | value |
      | market.value.windowLength                   | 60s   |
      | network.markPriceUpdateMaximumFrequency     | 0s    |
      | limits.markets.maxPeggedOrders              | 6     |
      | market.auction.minimumDuration              | 1     |
      | market.fee.factors.infrastructureFee        | 0.001 |
      | market.fee.factors.makerFee                 | 0.004 |
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

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA        |
      | ETH/MAR23 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA        |

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.5   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |

    Given the average block duration is "2"
  @Now
  Scenario: 001: lp1 and lp2 under supplies liquidity
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 50000             | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 10000             | 0.015 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 12        | 1                    | buy  | BID              | 12     | 20     | lp-b-1    |
      | lp1   | ETH/MAR22 | 12        | 1                    | sell | ASK              | 12     | 20     | lp-s-1    |
      | lp2   | ETH/MAR22 | 6         | 1                    | buy  | BID              | 6      | 20     | lp-b-2    |
      | lp2   | ETH/MAR22 | 6         | 1                    | sell | ASK              | 6      | 20     | lp-s-2    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 45976          | 1             |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 1 x 1 x 3.5569036

    And the liquidity fee factor should be "0.015" for the market "ETH/MAR22"

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | lp1   | ETH/MAR22 | 42683       | 51219  | 64024   | 72561   |
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 64024  | 0       | 35976 |
      | lp2   | USD   | ETH/MAR22 | 32013  | 57987   | 10000 |
    #margin_intial lp1: 12*1000*3.5569036*1.5=64024
    Then the network moves ahead "6" blocks
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 64024  | 0       | 17988 |
      | lp2   | USD   | ETH/MAR22 | 32013  | 57987   | 5000  |

    #AC: 0044-LIME-075, lp commit in multi markets
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_3 | lp2   | ETH/MAR23 | 500               | 0.02 | submission |

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp2   | USD   | ETH/MAR22 | 32013  | 57487   | 5000 |
      | lp2   | USD   | ETH/MAR23 | 0      | 57487   | 500  |

  @SLABug
  Scenario: 002: lp1 and lp2 amend LP commitment
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | USD   | 1000000 |
      | lp2    | USD   | 1000000 |
      | party1 | USD   | 100000  |
      | party2 | USD   | 100000  |
      | party3 | USD   | 100000  |
      | ptbuy  | USD   | 1000000 |
      | ptsell | USD   | 1000000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 50000             | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 10000             | 0.01 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 120       | 1                    | buy  | BID              | 120    | 20     | lp-b-1    |
      | lp1   | ETH/MAR22 | 120       | 1                    | sell | ASK              | 120    | 20     | lp-s-1    |
      | lp2   | ETH/MAR22 | 60        | 1                    | buy  | BID              | 60     | 20     | lp-b-2    |
      | lp2   | ETH/MAR22 | 60        | 1                    | sell | ASK              | 60     | 20     | lp-s-2    |
    Then the network moves ahead "2" blocks
    And the orders should have the following status:
      | party | reference | status        |
      | lp1   | lp-b-1    | STATUS_PARKED |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      #Then the network moves ahead "2" blocks

    Then the opening auction period ends for market "ETH/MAR22"

    And the orders should have the following status:
      | party | reference | status        |
      | lp1   | lp-b-1    | STATUS_ACTIVE |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 640243 | 309757  | 50000 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3556         | 60000          | 1             |

    #AC: 0044-LIME-018, lp reduces commitment
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 30000             | 0.02 | amendment |
    And the supplied stake should be "60000" for the market "ETH/MAR22"
    Then the network moves ahead "10" blocks
    And the supplied stake should be "40000" for the market "ETH/MAR22"

    #AC: 0044-LIME-019, lp reduces commitment multi times
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 28000             | 0.02 | amendment |
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 27000             | 0.02 | amendment |
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 27000             | 0.02 | amendment |
    And the supplied stake should be "40000" for the market "ETH/MAR22"
    Then the network moves ahead "10" blocks
    And the supplied stake should be "37000" for the market "ETH/MAR22"
    #AC: 0044-LIME-022, lp reduces commitment, no penalty
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 640243 | 332757  | 27000 |

    #AC:0044-LIME-021, lp changes fee factor
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 50000             | 0.008 | amendment |
    And the liquidity fee factor should be "0.01" for the market "ETH/MAR22"
    Then the network moves ahead "10" blocks
    And the liquidity fee factor should be "0.008" for the market "ETH/MAR22"

    #AC: 0044-LIME-030, lp increases commitment and they do not have sufficient collateral in the settlement asset
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   | error                             |
      | lp_1 | lp1   | ETH/MAR22 | 600000            | 0.02 | amendment | commitment submission rejected, not enough stake |
    Then the network moves ahead "1" blocks
    And the supplied stake should be "60000" for the market "ETH/MAR22"

    #AC: 0044-LIME-031, lp increases commitment and they have sufficient collateral in the settlement asset
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 60000             | 0.02 | amendment |
    And the supplied stake should be "70000" for the market "ETH/MAR22"

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 640243 | 299757  | 60000 |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3556         | 70000          | 1             |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | buy  | 1      | 960   | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/MAR22 | sell | 1      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
    And the liquidity fee factor should be "0.008" for the market "ETH/MAR22"
    #liquidity fee collected: 5*1000*0.008=40

    #AC: 0044-LIME-020, lp decreases commitment and gets bond slashing
    #AC: 0044-LIME-049, at the end of the current epoch rewards/penalties are evaluated based on the balance of the bond account at start of epoch
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 640243 | 299757  | 60000 |
      | lp2   | USD   | ETH/MAR22 | 320122 | 669878  | 10000 |
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | ETH/MAR22 | 1000              | 0.02 | amendment |
      | lp_2 | lp2   | ETH/MAR22 | 500               | 0.02 | amendment |
    Then the network moves ahead "1" blocks
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 640243 | 309757  | 50000 |
      | lp2   | USD   | ETH/MAR22 | 320122 | 669878  | 10000 |
    # trigger price auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | ptsell | ETH/MAR22 | sell | 1      | 940   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptbuy  | ETH/MAR22 | buy  | 1      | 940   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 0      | 950000  | 50000 |
      | lp2   | USD   | ETH/MAR22 | 0      | 990000  | 10000 |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 20274        | 60000          | 6             | 3           |
    When the network moves ahead "10" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 608232 | 386874  | 1002 |
      | lp2   | USD   | ETH/MAR22 | 304116 | 694629  | 501  |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest | horizon | min bound | max bound |
      | 950        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 20274        | 1503           | 6             | 3600    | 925       | 976       |

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 33     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 6      | USD   |
