Feature: vAMM rebasing when created or amended

  Background:
    Given the average block duration is "1"
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.00984363574304481    | 0.009937604878885509 | -1          | -1            | 0.2                    |
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

    And the following assets are registered:
      | id  | decimal places |
      | USD | 18             |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params | position decimal places | decimal places | allowed empty amm levels | linear slippage factor |
      | ETH/MAR22 | USD        | USD   | lqm-params           | my-simple-risk-model  | margin-calculator-1 | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         | SLA-22     | 3                       | 2              | 100                      | 0.001                  |

    # Setting up the accounts and vAMM submission now is part of the background, because we'll be running scenarios 0090-VAMM-006 through 0090-VAMM-014 on this setup
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount                    |
      | lp1    | USD   | 1000000000000000000000000 |
      | lp2    | USD   | 1000000000000000000000000 |
      | lp3    | USD   | 1000000000000000000000000 |
      | party1 | USD   | 1000000000000000000000000 |
      | party2 | USD   | 1000000000000000000000000 |
      | party3 | USD   | 1000000000000000000000000 |
      | party4 | USD   | 1000000000000000000000000 |
      | party5 | USD   | 1000000000000000000000000 |
      | vamm1  | USD   | 1000000000000000000000000 |
      | vamm2  | USD   | 1000000000000000000000000 |
      | vamm3  | USD   | 1000000000000000000000000 |
      | vamm4  | USD   | 1000000000000000000000000 |

  @VAMM @NoPerp
  Scenario: two crossed AMMs at opening auction end where orderbook shape refining is required

    # 39.98867525723519 11.328128751075417

    Then the parties submit the following AMM:
      | party | market id | amount                  | slippage | base    | upper bound    | proposed fee | lower leverage    | upper leverage      |
      | vamm1 | ETH/MAR22 | 2852341107003410000000  | 0.05     | 218564  | 281710         | 0.03         | 39.98867525723519 | 11.328128751075417  |
    Then the AMM pool status should be:
      | party | market id | amount                 | status        | base    | upper bound | lower leverage    | upper leverage      |
      | vamm1 | ETH/MAR22 | 2852341107003410000000 | STATUS_ACTIVE | 218564  | 281710      | 39.98867525723519 | 11.328128751075417  |

    And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0                | 0                 |
   
    Then the parties submit the following AMM:
      | party | market id | amount                  | slippage | base   | lower bound | upper bound | proposed fee | lower leverage   | upper leverage | 
      | vamm2 | ETH/MAR22 | 8514633449978613000000  | 0.05     |  372056 | 172861     | 452663      | 0.03         | 87.09361695065867 | 92.95166117996257 |
    Then the AMM pool status should be:
      | party | market id | amount                 | status        | base     | lower bound |  upper bound |lower leverage   | upper leverage | 
      | vamm2 | ETH/MAR22 | 8514633449978613000000 | STATUS_ACTIVE | 372056  | 172861      | 452663      | 87.09361695065867 | 92.95166117996257 |

    
    And set the following AMM sub account aliases:
      | party | market id | alias    |
      | vamm1 | ETH/MAR22 | vamm1-id |
      | vamm2 | ETH/MAR22 | vamm2-id |



    And the parties place the following orders:
      | party    | market id | side   | volume   | price    | resulting trades | type       | tif     |
      |  party1  | ETH/MAR22 |  buy   |  1       |  2       | 0                | TYPE_LIMIT | TIF_GTC |
      |  party1  | ETH/MAR22 |  buy   |  1       |  2       | 0                | TYPE_LIMIT | TIF_GTC |
      |  party2  | ETH/MAR22 |  buy   |  26      |  398908  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party2  | ETH/MAR22 |  buy   |  20      |  400384  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  36825   |  400600  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  35099   |  400602  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  33454   |  400604  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  31886   |  400606  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  30392   |  400608  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  28967   |  400610  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  27610   |  400612  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  26316   |  400614  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party2  | ETH/MAR22 |  buy   |  21      |  400616  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  25082   |  400616  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  buy   |  23907   |  400618  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party4  | ETH/MAR22 |  buy   |  250000  |  402896  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party4  | ETH/MAR22 |  buy   |  250000  |  402904  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party4  | ETH/MAR22 |  buy   |  250000  |  403052  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party4  | ETH/MAR22 |  buy   |  250000  |  403682  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party4  | ETH/MAR22 |  buy   |  250000  |  403916  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party4  | ETH/MAR22 |  buy   |  250000  |  404624  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  36823   |  400638  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  35097   |  400636  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  33452   |  400634  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  31885   |  400632  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  30390   |  400630  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  28966   |  400628  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  27608   |  400626  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  26315   |  400624  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  25081   |  400622  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party3  | ETH/MAR22 |  sell  |  23906   |  400620  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party2  | ETH/MAR22 |  sell  |  25      |  399916  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party2  | ETH/MAR22 |  sell  |  23      |  399688  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party2  | ETH/MAR22 |  sell  |  24      |  399064  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party5  | ETH/MAR22 |  sell  |  250000  |  396612  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party5  | ETH/MAR22 |  sell  |  250000  |  395916  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party5  | ETH/MAR22 |  sell  |  250000  |  395688  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party5  | ETH/MAR22 |  sell  |  250000  |  395072  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party5  | ETH/MAR22 |  sell  |  250000  |  394926  | 0                | TYPE_LIMIT | TIF_GTC |
      |  party5  | ETH/MAR22 |  sell  |  250000  |  394920  | 0                | TYPE_LIMIT | TIF_GTC |


    # now place some pegged orders which will cause a panic if the uncrossing is crossed
    When the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | lp3   | ETH/MAR22 | buy  | 100    | BID              | 1      |
      | lp3   | ETH/MAR22 | sell | 100    | ASK              | 1      |

   And the network moves ahead "1" blocks

   And the market data for the market "ETH/MAR22" should be:
      | trading mode                 | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 400267              | 1511468                |

   When the opening auction period ends for market "ETH/MAR22"
      
    Then the network moves ahead "1" blocks

    # two AMMs are now prices at ~100 which is between their base values
    And the market data for the market "ETH/MAR22" should be:
      | mark price    | trading mode             |
      | 400618        | TRADING_MODE_CONTINUOUS  |
