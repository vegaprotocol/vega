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
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params | allowed empty amm levels |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | default-none     | termination-oracle     | 1e0                    | 0                         | SLA-22     | 5                        |

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
      | vamm3  | USD   | 1000000 |


    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 20     | 40    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party5 | ETH/MAR22 | buy  | 20     | 90    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |


    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 95          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 95          | 105         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 103             | 92                |

  @VAMM
  Scenario: Incoming order at AMM best price

  # AMM's has a BUY at 99 so a SELL at that price should match
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 1      | 99    | 1                | TYPE_LIMIT | TIF_GTC |           |
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | vamm1-id  | 99    | 1    | party1    | true   |

  @VAMM
  Scenario: Incoming order at AMM best price and orderbook volume exists at that price

  # AMM's has a BUY at 99 so a SELL at that price should match
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | buy  | 100    | 99    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | sell | 150    | 99    | 2                | TYPE_LIMIT | TIF_GTC |           |

  # incoming move AMM to a fair-price of 99, then we take the orderbook volume at 99
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | vamm1-id  | 99    | 103  | party1    | true   |
      | party2    | 99    | 47   | party1    | true   |


  @VAMM
  Scenario: Incoming order at AMM best price and orderbook volume exists at FAIR PRICE

  # AMM's has a BUY at 99 so a SELL at that price should match
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | buy  | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | sell | 150    | 99    | 2                | TYPE_LIMIT | TIF_GTC |           |

  # incoming absorbs order at fair price, then we take volume from AMM
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | party2    | 100   | 100  | party1    | true   |
      | vamm1-id  | 99    | 50   | party1    | true   |


  @VAMM
  Scenario: Incoming order at AMM fair price and orderbook volume exists at FAIR PRICE

  # AMM's has a BUY at 99 so a SELL at 100 should not match
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | buy  | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | sell | 150    | 100   | 1                | TYPE_LIMIT | TIF_GTC |           |

  # incoming absorbs order at fair price, then we take volume from AMM
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | party2    | 100   | 100  | party1    | true   |

  @VAMM
  Scenario: Incoming order at AMM fair price and orderbook volume exists at FAIR PRICE and at AMM best price
  When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/MAR22 | buy  | 50     | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | buy  | 50     | 99    | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | ETH/MAR22 | sell | 200    | 99    | 3                | TYPE_LIMIT | TIF_GTC |           |

  # incoming absorbs order at fair price, then we take volume from AMM
  Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | party2    | 100   | 50   | party1    | true   |
      | vamm1-id  | 99    | 103  | party1    | true   |
      | party2    | 99    | 47   | party1    | true   |


  @VAMM
  Scenario: AMM's cannot be submitted/amended/cancelled when a market is terminated
  
      When the oracles broadcast data signed with "0xCAFECAFE19":
      | name               | value |
      | trading.terminated | true  |
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | trading mode            |
      | TRADING_MODE_NO_TRADING |

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee | error               |
      | vamm2 | ETH/MAR22 | 100000  | 0.05    | 100  | 95          | 105         | 0.03         | trading not allowed |

    Then the parties amend the following AMM:
       | party  | market id | amount | slippage | base | lower bound | upper bound | upper leverage | error               |
       | vamm1  | ETH/MAR22 | 20000  | 0.15     | 1010 | 910         | 1110        | 0.2            | trading not allowed |

    Then the parties cancel the following AMM:
       | party  | market id | method           | error               |
       | party1 | ETH/MAR22 | METHOD_IMMEDIATE | trading not allowed |


  @VAMM
  Scenario: Cannot amend AMM to have only one side if its current position does not support it
  
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 1      | 99    | 1                | TYPE_LIMIT | TIF_GTC |           |


    Then the parties amend the following AMM:
      | party | market id | amount | slippage | base | upper bound | proposed fee | error                                      |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 105         | 0.03         | cannot remove lower bound when AMM is long |


    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 2      | 200   | 2                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties amend the following AMM:
      | party | market id | amount | slippage | base | lower bound | proposed fee | error                                       |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 95          | 0.03         | cannot remove upper bound when AMM is short |


  @VAMM3
  Scenario: Two AMM's incoming order split pro-rata equally

    When the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000  | 0.05     | 100  | 95          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 95          | 105         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm2 | ETH/MAR22 | vamm2-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 206             | 184               |


    When the parties place the following orders:
        | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference |
        | party1 | ETH/MAR22 | sell | 1000    | 10    | 2                | TYPE_LIMIT | TIF_GTC |           |
    Then the following trades should be executed:
        | buyer     | price | size | seller    | is amm |
        | vamm1-id  | 98    | 500  | party1    | true   |
        | vamm2-id  | 98    | 500  | party1    | true   |
   
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 95             | 96               | 70              | 150               |

  @VAMM
  Scenario: Two AMM's incoming order split pro-rata unequally

    When the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 75000   | 0.05     | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 75000  | STATUS_ACTIVE | 100  | 90          | 110         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm2 | ETH/MAR22 | vamm2-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 141             | 125               |


    When the parties place the following orders:
        | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference |
        | party1 | ETH/MAR22 | sell | 400     | 10    | 2                | TYPE_LIMIT | TIF_GTC |           |
    Then the following trades should be executed:
        | buyer     | price | size | seller    | is amm |
        | vamm1-id  | 98    | 292  | party1    | true   |
        | vamm2-id  | 98    | 108  | party1    | true   |
   
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 96             | 98               | 184             | 113               |


  @VAMM
  Scenario: Two AMM's incoming order split pro-rata through base price where one side is much wider

    When the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000  | 0.05     | 100  | 95          | 500         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 95          | 500         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm2 | ETH/MAR22 | vamm2-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 206             | 92                |


    When the parties place the following orders:
        | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference |
        | party1 | ETH/MAR22 | sell | 400     | 10    | 2                | TYPE_LIMIT | TIF_GTC |           |
    Then the following trades should be executed:
        | buyer     | price | size | seller    | is amm |
        | vamm1-id  | 99    | 200  | party1    | true   |
        | vamm2-id  | 99    | 200  | party1    | true   |
   
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 97             | 99               | 232             | 194               |


    # now lets swing back across both AMM's base price, AMM2 will have much less volume on that side 
    When the parties place the following orders:
        | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference |
        | party1 | ETH/MAR22 | buy  | 600     | 115   | 4                | TYPE_LIMIT | TIF_GTC |           |
    Then the following trades should be executed:
        | seller     | price | size | buyer    | is amm |
        | vamm1-id   | 98    | 200  | party1   | true   |
        | vamm2-id   | 98    | 200  | party1   | true   |
        | vamm1-id   | 101   | 199  | party1   | true   |
        | vamm2-id   | 100   | 1    | party1   | true   |
    
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 102            | 104              | 16              | 163               |



@VAMM
  Scenario: 2 AMM's incoming order split pro-rata through, two AMM's are low volume on opposite ends

    When the parties amend the following AMM:
       | party  | market id | amount  | slippage | base | lower bound | upper bound |   
       | vamm1  | ETH/MAR22 | 500000  | 0.15     | 2000 | 1995        | 2300        |


    When the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 500000  | 0.05     | 2000 | 1700        | 2005        | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 500000 | STATUS_ACTIVE | 2000  | 1700       | 2005        | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm2 | ETH/MAR22 | vamm2-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 1999           | 2001             | 25              | 23                |


    When the parties place the following orders:
        | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference |
        | party1 | ETH/MAR22 | sell | 100     | 10    | 2                | TYPE_LIMIT | TIF_GTC |           |
    Then the following trades should be executed:
        | buyer     | price  | size | seller    | is amm |
        | vamm1-id  | 1998   | 99   | party1    | true   |
        | vamm2-id  | 1998   | 1    | party1    | true   |
   
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 1995           | 1997             | 30              | 22                |


    # now lets swing back across both AMM's base price, AMM2 will have much less volume on that side 
    When the parties place the following orders:
        | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference |
        | party1 | ETH/MAR22 | buy  | 200     | 5000  | 4                | TYPE_LIMIT | TIF_GTC |           |
    Then the following trades should be executed:
        | seller    | price | size | buyer     | is amm |
        | vamm1-id  | 1997  | 99   | party1    | true   |
        | vamm2-id  | 1998  | 1    | party1    | true   |
        | vamm1-id  | 2001  | 1    | party1    | true   |
        | vamm2-id  | 2002  | 99   | party1    | true   |
   
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 2004           | 2005             | 5               | 19                |