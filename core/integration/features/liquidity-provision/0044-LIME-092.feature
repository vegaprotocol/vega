Feature: Test LP mechanics when there are multiple liquidity providers;

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001 | 0.1 | 0 | 0 | 1.0 |
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
      | 0.00001 | 0.5 | 1 | 1.0 |

    And the liquidity sla params named "SLA2":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5 | 0.5 | 1 | 1.0 |

    And the liquidity sla params named "SLA3":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.05        | 1                            | 1                             | 1.0                    |

    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 10               | 3600s       | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                     | 60s   |
      | market.stake.target.timeWindow                | 20s   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0.5   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | limits.markets.maxPeggedOrders                | 6     |
      | market.auction.minimumDuration                | 1     |
      | market.fee.factors.infrastructureFee          | 0.001 |
      | market.fee.factors.makerFee                   | 0.004 |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.5 |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.providersFeeCalculationTimeStep | 10s |

    And the markets:
      | id        | quote name | asset | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params | liquidity monitoring |
      | ETH/MAR22 | USD        | USD   | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA        | lqm-params           |
      | ETH/MAR23 | USD        | USD   | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA2       | lqm-params           |
      | ETH/JAN23 | USD        | USD   | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA3       | lqm-params           |

    Given the average block duration is "2"
  @Now

  Scenario: An LP with bid orders inside valid range during auction (and market has no indicative price), is not penalised (0044-LIME-092)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 20000000   |
      | party1 | USD   | 1000000000 |
      | party2 | USD   | 1000000000 |
      | party3 | USD   | 1000000    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/JAN23 | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | ETH/JAN23 | buy  | 100    | 4750  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/JAN23 | sell | 100    | 5250  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/JAN23"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 17784        | 180000         | 1             |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 5000   | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 5000   | 5000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | 3600    | 4865      | 5139      | 88940284     | 180000         | 5001          |
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond   |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17518938 | 180000 |

    When the network moves ahead "2" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond   |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17518938 | 180000 |

  Scenario: An LP with ask orders outside valid range during auction is penalised (0044-LIME-094)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 20000000   |
      | party1 | USD   | 1000000000 |
      | party2 | USD   | 1000000000 |
      | party3 | USD   | 1000000    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/JAN23 | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | ETH/JAN23 | buy  | 100    | 3790  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/JAN23 | sell | 100    | 5250  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/JAN23"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 17784        | 180000         | 1             |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 5000   | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 5000   | 5000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | 3600    | 4865      | 5139      | 88940284     | 180000         | 5001          |
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 4000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 4000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond   |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17018938 | 108000 |

    When the network moves ahead "2" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond  |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17018938 | 27000 |

    Then the following transfers should happen:
      | from | to     | from account      | to account             | market id | amount | asset |
      | market | lp1    | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/JAN23 | 500000 | USD |
      | lp1    | market | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_INSURANCE         | ETH/JAN23 | 500000 | USD |
      | lp1    | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE         | ETH/JAN23 | 72000  | USD |
      | lp1    | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE         | ETH/JAN23 | 54000  | USD |
      | lp1    | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE         | ETH/JAN23 | 27000  | USD |







