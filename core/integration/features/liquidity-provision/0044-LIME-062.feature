Feature: Test changing market.liquidity.providersFeeCalculationTimeStep;

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 60s   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 6     |
      | market.auction.minimumDuration                      | 1     |
      | market.fee.factors.infrastructureFee                | 0.001 |
      | market.fee.factors.makerFee                         | 0.004 |
      | market.liquidity.providersFeeCalculationTimeStep    | 10s   |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.1   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 15s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.25  |
      | market.liquidity.equityLikeShareFeeFraction         | 1     |

    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 20s         | 1.0            |

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

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-1      |

    Given the average block duration is "1"
  @Now
  Scenario: 001: lp1 and lp2 on the market ETH/MAR22, 0044-LIME-062
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | USD   | 100000  |
      | lp2    | USD   | 100000  |
      | party1 | USD   | 1000000 |
      | party2 | USD   | 1000000 |

    And the network moves ahead "1" blocks
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 6000              | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 4000              | 0.015 | submission |

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

    When the network moves ahead "4" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 10000          | 1             |


    Then the network moves ahead "1" epochs
    Then the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 3s    |

    # As we are still within the epoch, the change to 3s distributions should not occur yet, so we make a trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    # Then forward enough blocks that the 3s setting would have caused a distribution and place another trade
    Then the network moves ahead "4" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "8" blocks

    # Now forward enough that a distribution does occur at the original 10s frequency and see one took place with the
    # volume from both orders in one transfer, implying nothing happened between the two orders
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 36     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 24     | USD   |

    # We are now nearly at the epoch boundary, so place an order just before
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs
    # And once we pass the epoch, this is distributed to the fee accounts at the new frequency
    And the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 18     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 12     | USD   |

    # So now if we place a new trade
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 3      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 3      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |
    Then the network moves ahead "4" blocks
    # Forward past the distribution frequency cutoff and place another
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 4      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | buy  | 4      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    # Then forward enough to cause them both to distribute, the fees are instead sent as two separate distributions
    Then the network moves ahead "7" blocks
    And the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 27     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 18     | USD   |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 48     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 32     | USD   |

  @Now
  Scenario: 001b: lp1 and lp2 on the market ETH/MAR22, 0044-LIME-062 with ELS fee share fraction set to 0.5
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | USD   | 100000  |
      | lp2    | USD   | 100000  |
      | party1 | USD   | 1000000 |
      | party2 | USD   | 1000000 |
    And the following network parameters are set:
      | name                                        | value |
      | market.liquidity.equityLikeShareFeeFraction | 0.5   |

    And the network moves ahead "1" blocks
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 6000              | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 4000              | 0.015 | submission |

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

    When the network moves ahead "4" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 10000          | 1             |


    Then the network moves ahead "1" epochs
    Then the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 3s    |

    # As we are still within the epoch, the change to 3s distributions should not occur yet, so we make a trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    # Then forward enough blocks that the 3s setting would have caused a distribution and place another trade
    Then the network moves ahead "4" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "8" blocks

    # Now forward enough that a distribution does occur at the original 10s frequency and see one took place with the
    # volume from both orders in one transfer, implying nothing happened between the two orders
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 33     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 27     | USD   |

    # We are now nearly at the epoch boundary, so place an order just before
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs
    # And once we pass the epoch, this is distributed to the fee accounts at the new frequency
    And the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 16     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 13     | USD   |

    # So now if we place a new trade
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 3      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 3      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |
    Then the network moves ahead "4" blocks
    # Forward past the distribution frequency cutoff and place another
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 4      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | buy  | 4      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |           |

    # Then forward enough to cause them both to distribute, the fees are instead sent as two separate distributions
    Then the network moves ahead "7" blocks
    And the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 25     | USD   |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 44     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 20     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 36     | USD   |
