Feature: vAMM with oracle driven base price

  Background:
    Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5             |
      | USD | 0              |
    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | source weights | source staleness tolerance | spec id |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 5                   | 1,0,0,0        | 100s,0s,0s,0s              | 1234    |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.01             | 10s         | 5              |  

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params | tick size |
      | ETH/MAR22 | ETH        | ETH   | lqm-params           | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 0              | 5                       | perp        | SLA        |     1     |
    And the following network parameters are set:
      | name                                             | value |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.auction.minimumDuration                   | 1     |
      | market.fee.factors.infrastructureFee             | 0.001 |
      | market.fee.factors.makerFee                      | 0.004 |
      | market.value.windowLength                        | 60s   |
      | market.liquidity.bondPenaltyParameter            | 0.1   |
      | validators.epoch.length                          | 5s    |
      | limits.markets.maxPeggedOrders                   | 2     |
      | market.liquidity.providersFeeCalculationTimeStep | 5s    |
      | market.liquidity.stakeToCcyVolume                | 1     |

    And the average block duration is "1"
    # Setting up the accounts and vAMM submission now is part of the background, because we'll be running scenarios 0090-VAMM-006 through 0090-VAMM-014 on this setup
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | ETH   | 100000000000 |
      | lp2    | ETH   | 100000000000 |
      | lp3    | ETH   | 100000000000 |
      | party1 | ETH   | 100000000000 |
      | party2 | ETH   | 100000000000 |
      | party3 | ETH   | 100000000000 |
      | party4 | ETH   | 100000000000 |
      | party5 | ETH   | 100000000000 |
      | vamm1  | ETH   | 100000000000 |
      | vamm2  | ETH   | 100000000000 |
      | vamm3  | ETH   | 100000000000 |


    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |           |
    When the opening auction period ends for market "ETH/MAR22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 100   | 1    | party2 |


  @VAMM
  Scenario: 0090-VAMM-038 It's possible to setup the vAMM so that it uses one of the oracles already available for the market in which it operates for its `base price`. In that case the deployment attempt should be deferred until the next value is received from the oracle.

   Then the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee | data source id |
      | vamm1 | ETH/MAR22 | 100000  | 0.05     | 0    | 95          | 105         | 0.03         | 1234           |
    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_PENDING | 0    | 95          | 105         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 0              | 0                | 0               | 0                 |

    # now oracle data comes in for the perp and it drives the AMM's base price
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 10000000   | -2s         |

    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE  | 100  | 95          | 105         | 

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |

    # now oracle data comes in for a price that is outside of the lower bound the AMM doesn't get updated
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 9000000   | -1s        |

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |

    # now oracle data comes in for a price that is outside of the upper bound the AMM doesn't get updated
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 20000000 | 0s          |

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |

      # now oracle data comes in for a price that is outside of the upper bound the AMM doesn't get updated
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 9900000    | 1s          |

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 98             | 100              | 1260            | 764               |

  @VAMM
  Scenario: 0090-VAMM-039 AMM has a static base price and amends itself to a oracle drive base price
   Then the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05     | 100  | 95          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE  | 100  | 95          | 105         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |

    # now oracle data comes in nothing changes on the AMM
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 10000000   | -2s         |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |

    # the base price is inherited from the old AMM
    Then the parties amend the following AMM:
       | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee | data source id |
       | vamm1 | ETH/MAR22 | 100000  | 0.05     | 0    | 95          | 105         | 0.03         | 1234           |
    Then the AMM pool status should be:
      | party | market id | amount | status          | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE   | 100  | 95          | 105         | 

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |

    # now it can get updated
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 9900000    | -1s          |

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 98             | 100              | 1260            | 764               |

  @VAMM
  Scenario: AMM has a static base price and amends itself to a oracle drive base price where old base it out of range
   Then the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.05     | 100  | 95          | 105         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE  | 100  | 95          | 105         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |

    # the base price is inherited from the old AMM
    Then the parties amend the following AMM:
       | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee | data source id |
       | vamm1 | ETH/MAR22 | 1000000 | 0.05     | 0    | 200         | 210         | 0.03         | 1234           |
    Then the AMM pool status should be:
      | party | market id | amount  | status          | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 1000000 | STATUS_PENDING  | 0    | 200         | 210         | 

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 0              | 0                | 0               | 0                 |

    # now it can get updated
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 20500000    | -1s          |

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 204            | 206              | 5037            | 4871              |


  @VAMM
  Scenario: 0090-VAMM-040, 0090-VAMM-041 base only gets update if difference in price is less that minimum price change trigger
   Then the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee | data source id | minimum price change trigger |
      | vamm1 | ETH/MAR22 | 100000  | 0.05     | 0    | 95          | 105         | 0.03         | 1234           | 0.02                         |
    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_PENDING | 0    | 95          | 105         | 

    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 0              | 0                | 0               | 0                 |

    # now oracle data comes in and update base price, the AMM is pending so will take anything
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 10000000   | -2s         |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |


    Then the network moves ahead "1" blocks
    # oracle data too close in price, no update
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 10100000   | -1s         |
    
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 976             | 931               |


    # oracle data shifts the base price more than the minimum, update happens
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 10300000   | 0s          |
    
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 102            | 104              | 556             | 2447              |

  
  @VAMM
  Scenario: 0090-VAMM-042 updated base price is good, but slippage to rebase prevents it

   Then the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee | data source id |
      | vamm1 | ETH/MAR22 | 100000  | 0.02     | 0    | 80          | 120         | 0.03         | 1234           |
    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_PENDING | 0    | 80          | 120         | 

    # now oracle data comes in for the perp and it drives the AMM's base price
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 10000000   | -2s         |

    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE  | 100  | 80          | 120         | 

    # now we place a big order that doesn't cross
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 200    | 101   | 0                | TYPE_LIMIT | TIF_GTC |           |


    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 177             | 373               |


    # now oracle data comes in for the perp and it drives the AMM's base price
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 11500000   | -1s         |

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 177             | 373               |

     # now oracle data comes in for the perp and it drives the AMM's base price
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 10200000   | 0s         |

    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE  | 102  | 80          | 120         | 

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 100            | 101              | 155             | 47                |


  @VAMM
  Scenario: 0090-VAMM-043 AMM amendment uses slippage straight away
   Then the parties submit the following AMM:
      | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee |
      | vamm1 | ETH/MAR22 | 100000  | 0.01     | 100  | 80          | 120         | 0.03         |
    Then the AMM pool status should be:
      | party | market id | amount | status         | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE  | 100  | 80          | 120         | 


    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 177             | 173               |

    # the base price is inherited from the old AMM
    Then the parties amend the following AMM:
       | party | market id | amount  | slippage | base | lower bound | upper bound | proposed fee | data source id |
       | vamm1 | ETH/MAR22 | 100000  | 0.05     | 0    | 80          | 120         | 0.03         | 1234           |
    Then the AMM pool status should be:
      | party | market id | amount | status          | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE   | 100  | 80          | 120         | 


     # now we place a big order that doesn't cross
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 700    | 102   | 0                | TYPE_LIMIT | TIF_GTC |           |


    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode             | best bid price | best offer price | best bid volume | best offer volume |
      | 100        | TRADING_MODE_CONTINUOUS  | 99             | 101              | 177             | 173               |

    # now it can get updated
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 11500000   | -1s         |

    # base price update only happened because the slippage was increased in the amend that set the new oracle value
    Then the AMM pool status should be:
      | party | market id | amount | status          | base | lower bound | upper bound | 
      | vamm1 | ETH/MAR22 | 100000 | STATUS_ACTIVE   | 115  | 80          | 120         | 