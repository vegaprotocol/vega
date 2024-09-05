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
      | vamm2  | USD   | 1000000 |
      | vamm3  | USD   | 1000000 |
      | vamm4  | USD   | 1000000 |

  @VAMM
  Scenario: two crossed AMMs at opening auction end

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 102  | 92          | 112         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 102  | 92          | 112         | 

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0                | 0                 |
   
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000  | 0.05    | 98   | 88          | 108         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE |  98  | 88          | 108         | 

    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |

   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 100              | 91                |

   When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | vamm1-id  | 100   | 46   | vamm2-id  | true   |
      
    Then the network moves ahead "1" blocks

    # two AMMs are now prices at ~100 which is between their base values
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              |


  @VAMM
  Scenario: two AMM's that cross at a single point i.e no overlap

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.05     | 100  | 95          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 95          | 105         | 

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0                | 0                 |
   
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base  | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000  | 0.05    | 102   | 97          | 107         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 102  | 97          | 107         | 

    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |

   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 101              | 92                |

   When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | vamm2-id  | 101   | 92   | vamm1-id  | true   |
      
    Then the network moves ahead "1" blocks

    # two AMMs are now prices at ~100 which is between their base values
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 101        | TRADING_MODE_CONTINUOUS  | 100            | 102              |

  @VAMM
  Scenario: AMM crossed with SELL orders

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         | 


    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | sell | 100    | 95    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |


   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 96               | 100               |

   When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | vamm1-id  | 96    | 100  | lp1       | true   |

      
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 96         | TRADING_MODE_CONTINUOUS  | 97             | 99               |

  @VAMM
  Scenario: AMM crossed with BUY orders

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         | 

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 100    | 105   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |


   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 104              | 100               |

   When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | lp1       | 104   | 100  | vamm1-id  | true   |

      
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 104        | TRADING_MODE_CONTINUOUS  | 102            | 104              |


  @VAMM
  Scenario: AMM's crossed with orders and AMMs

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.05     | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         | 

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000 | 0.05     | 98   | 88          | 108         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 98   | 88          | 108         | 

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 50     | 105   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | buy  | 50     | 102   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp2    | ETH/MAR22 | sell | 50     | 95    | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
      | lp2    | ETH/MAR22 | sell | 50     | 98    | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |


   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 99               | 146               |

   When the opening auction period ends for market "ETH/MAR22"
   Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | lp1       | 99    | 46   | vamm2-id  | true   |
      | lp1       | 99    | 4    | lp2       | false  |
      | lp1       | 99    | 46   | lp2       | false  |
      | lp1       | 99    | 4    | lp2       | false  |
      | vamm1-id  | 99    | 46   | lp2       | true   |

      
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 99         | TRADING_MODE_CONTINUOUS  | 98             | 100              |


  @VAMM
  Scenario: Crossed orders then AMM submitted

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 50     | 105   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | buy  | 50     | 102   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp2    | ETH/MAR22 | sell | 50     | 95    | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
      #| lp2    | ETH/MAR22 | sell | 50     | 98    | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 100              | 50               |

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         | 

    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |


   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 102              | 100               |

   When the opening auction period ends for market "ETH/MAR22"
    #Then the following trades should be executed:
    #  | buyer     | price | size | seller    | is amm |
    #  | vamm1-id  | 96    | 100  | lp1       | true   |

      
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 102        | TRADING_MODE_CONTINUOUS  | 101            | 103              |

  @VAMM
  Scenario: AMM cancelled and amending when in auction

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         | 

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0                | 0               |


    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000  | 0.05    | 95   | 85          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 95   | 85          | 105         | 

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 98               | 104               |


    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |


    # amend so that its not crossed
    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 0.1      | 100  | 90          | 110         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         |

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0                | 0                 |


    # amend so that its more crossed again at a different point
    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 0.1      | 98  | 88          | 108         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 98   | 88          | 108         |

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 99               | 46                |

    # then the second AMM is cancels
    When the parties cancel the following AMM:
      | party | market id | method             |
      | vamm2 | ETH/MAR22 | METHOD_IMMEDIATE   |

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0                | 0                 |
    

    # then amend the first AMM and re-create the second
    When the parties amend the following AMM:
      | party | market id | slippage | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 0.1      | 98  | 88          | 108         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 98   | 88          | 108         |

    When the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000  | 0.05    | 102  | 92          | 112         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 102   | 92         | 112         | 


    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 100              | 91                |


    # now uncross
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer     | price | size | seller    | is amm |
      | vamm2-id  | 100   | 46   | vamm1-id  | true   |

      
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              |

  Scenario: Stagnet auction uncrossing panic where uncrossing side has both AMM and orderbook volume at the same level but we only need the orderbook volume
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05    | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         | 


    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | sell | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | sell | 160    | 95    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp2    | ETH/MAR22 | buy  | 2      | 98    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |

   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume | best bid price | best offer price |
      | TRADING_MODE_OPENING_AUCTION | 96               | 160               | 99             | 95               |

    
    When the opening auction period ends for market "ETH/MAR22"
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price   | trading mode               | best bid price | best offer price |
      | 96           | TRADING_MODE_CONTINUOUS    | 96             | 98               |

@VAMM
  Scenario: complicated AMM's crossed with orders and pegged orders

  # these are buys
  And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1    | ETH/MAR22 | buy  | 50     | 105   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | buy  | 50     | 103   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | buy  | 50     | 101   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | buy  | 50     | 99    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp1    | ETH/MAR22 | buy  | 50     | 98    | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |

  # these are sells
  And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp2    | ETH/MAR22 | sell  | 50    | 102  | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
      | lp2    | ETH/MAR22 | sell  | 50    | 98   | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
      | lp2    | ETH/MAR22 | sell  | 50    | 95   | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |


    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.05     | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         | 

    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm2 | ETH/MAR22 | 100000 | 0.05     | 90   | 85          | 95         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm2 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 90   | 85          | 95         | 


    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm3 | ETH/MAR22 | 100000 | 0.05     | 90   | 85          | 95         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound | 
      | vamm3 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 90   | 85          | 95          | 



    # now place some pegged orders which will cause a panic if the uncrossing is crossed
    When the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | lp3   | ETH/MAR22 | buy  | 100    | BID              | 1      |
      | lp3   | ETH/MAR22 | sell | 100    | ASK              | 1      |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |


    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 93               | 602               |


    When the opening auction period ends for market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 93         | TRADING_MODE_CONTINUOUS  | 92             | 94               |


  @VAMM
  Scenario: AMM crossed with limit order, AMM pushed to boundary


    And the parties place the following orders:
        | party  | market id | side  | volume | price | resulting trades | type       | tif     | reference |
        | lp1    | ETH/MAR22 | buy   | 423    | 200   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      
    Then the parties submit the following AMM:
      | party | market id | amount | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000 | 0.05     | 100  | 90          | 110         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status        | base | lower bound | upper bound |
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE | 100  | 90          | 110         |


    # now place some pegged orders which will cause a panic if the uncrossing is crossed
    When the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | lp3   | ETH/MAR22 | buy  | 100    | BID              | 1      |
      | lp3   | ETH/MAR22 | sell | 100    | ASK              | 1      |

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |


    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 155              | 423               |


    When the opening auction period ends for market "ETH/MAR22"

    # the volume of this trade should be the entire volume of the AMM's sell curve
    Then the following trades should be executed:
      | buyer     | price | size  | seller     | is amm |
      | lp1       | 155   | 423   | vamm1-id  | true   |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price |
      | 155        | TRADING_MODE_CONTINUOUS  | 109            | 0                |