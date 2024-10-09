Feature: vAMM rebasing when created or amended

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
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0   |
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

    And the oracle spec for settlement data filtering data from "0xCAFECAFE19" named "termination-oracle":
      | property         | type         | binding         | decimals |
      | prices.ETH.value | TYPE_INTEGER | settlement data | 0        |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE19" named "termination-oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | termination-oracle     | 1e0                    | 0                         | SLA-22     |

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
      | vamm2  | USD   | 1000000 |


    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party5 | ETH/MAR22 | buy  | 20     | 90    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party3 | ETH/MAR22 | sell | 10     | 130   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | ETH/MAR22 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |


    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 1000   | 0.05     | 100  | 90          | 120         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 1000 | STATUS_ACTIVE | 100  | 90          | 120         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |


    # note that the commitment is low and is creating zero volume price ticks so the best bid is more than 1 away from the AMM's
    # fair price
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 98             | 105              | 1               | 1                 |

  @VAMM
  Scenario: Sparse AMM incoming orders makes it long

  # AMM's has a BUY at 98 so this SELL order won't match with 99
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 5      | 99    | 0                | TYPE_LIMIT | TIF_GTC |           |
  
  And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 98             | 99               | 1               | 5                 |
 
  # lets send in a 98 that will now match with the AMM
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 1      | 98    | 1                | TYPE_LIMIT | TIF_GTC |           |
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | vamm1-id  | 99    | 1    | party1    | true   |

  # AMM is now quoting at 96 and 100, but we have our sell limit order at 99.
  And the market data for the market "ETH/MAR22" should be:
      | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | TRADING_MODE_CONTINUOUS  | 96             | 99               | 1               | 5                 |

  # lets send in a 99 to eat that limit order and leave us with AMM
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | buy  | 5      | 99    | 1                | TYPE_LIMIT | TIF_GTC |           |
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | party2    | 99    | 5    | party1    | true   |

  # AMM is now quoting at 96 and 100
  And the market data for the market "ETH/MAR22" should be:
      | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | TRADING_MODE_CONTINUOUS  | 96             | 100              | 1               | 1                 |
  
  # order at 99 doesn't match since the AMM's volume is at 100
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | buy  | 5      | 99    | 0                | TYPE_LIMIT | TIF_FOK |           |
  
  # this one will
  When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | buy  | 1      | 100    | 1                | TYPE_LIMIT | TIF_FOK |           |

  And the market data for the market "ETH/MAR22" should be:
      | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | TRADING_MODE_CONTINUOUS  | 98             | 105              | 1               | 1                 |


  @VAMM
  Scenario: Sparse AMM incoming orders makes it short

  # AMM's has a BUY at 98 so this SELL order won't match with 99
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 5      | 102   | 0                | TYPE_LIMIT | TIF_GTC |           |
  
  And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 102            | 105              | 5               | 1                 |
 
  # lets send in a 105 that will now match with the AMM
  When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 1      | 105    | 1                | TYPE_LIMIT | TIF_GTC |           |

  # AMM is now quoting at 100 and 110, but we have our sell limit order at 102.
  And the market data for the market "ETH/MAR22" should be:
      | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | TRADING_MODE_CONTINUOUS  | 102            | 111              | 5               | 1                 |

  # lets send in a 102 to eat that limit order and leave us with AMM
  When the parties place the following orders:
      | party  | market id | side  | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | sell  | 5      | 102    | 1               | TYPE_LIMIT | TIF_GTC |           |
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | party1    | 102   | 5    | party2    | true   |

  # AMM is now quoting at 96 and 100
  And the market data for the market "ETH/MAR22" should be:
      | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | TRADING_MODE_CONTINUOUS  | 100            | 111              | 1               | 1                 |
  
  # order at 102 doesn't match since the AMM's volume is at 100
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | sell | 5      | 102   | 0                | TYPE_LIMIT | TIF_FOK |           |
  
  # this one will
  When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | sell | 1      | 100    | 1                | TYPE_LIMIT | TIF_FOK |           |

  And the market data for the market "ETH/MAR22" should be:
      | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | TRADING_MODE_CONTINUOUS  | 98             | 105              | 1               | 1                 |