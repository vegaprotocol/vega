Feature: Test change of SLA market parameter

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

    And the liquidity sla params named "SLA-22-2":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.01 | 0.6 | 1 | 1.0 |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5 | 0.6 | 1 | 1.0 |

    And the markets:
      | id        | quote name | asset | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD | USD | log-normal-risk-model | margin-calculator-1 | 2 | fees-config-1 | price-monitoring | default-eth-for-future | 1e0 | 0 | SLA-22-2 |

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.7 |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |

    Given the average block duration is "1"
  @Now
  Scenario: 001: lp1 and lp2 on the market ETH/MAR22, 0044-LIME-092, 0044-LIME-094
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1 | USD | 200000 |
      | lp2 | USD | 200000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 4000              | 0.02  | submission |
      | lp_2 | lp2   | ETH/MAR22 | 4000              | 0.015 | submission |

    When the network moves ahead "1" blocks

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 10     | 910   | 0                | TYPE_LIMIT | TIF_GTC | best-buy-1  |
      | party1 | ETH/MAR22 | buy | 2 | 1000 | 0 | TYPE_LIMIT | TIF_GTC |  |
      | party2 | ETH/MAR22 | sell | 10     | 1110  | 0                | TYPE_LIMIT | TIF_GTC | best-sell-1 |
      | party2 | ETH/MAR22 | sell | 2 | 1000 | 0 | TYPE_LIMIT | TIF_GTC |  |

    Then the opening auction period ends for market "ETH/MAR22"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | best-buy-2  |
      | party2 | ETH/MAR22 | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | best-sell-2 |

    When the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 14227        | 8000           | 4             |

    Then the parties cancel the following orders:
      | party  | reference   |
      | party1 | best-buy-1  |
      | party2 | best-sell-1 |
    When the network moves ahead "1" blocks
#we do not have `indicative uncrossing price`.
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 14227        | 8000           | 4             |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR22 | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC | lp1--b    |
      | lp2   | ETH/MAR22 | buy  | 10     | 989   | 0                | TYPE_LIMIT | TIF_GTC | lp2-GFA-b |
      | lp2   | ETH/MAR22 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | lp2-GFA-s |
      | lp1   | ETH/MAR22 | sell | 10     | 1011  | 0                | TYPE_LIMIT | TIF_GTC | lp1-GFA-s |

    When the network moves ahead "1" blocks
    And the current epoch is "0"
    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    When the network moves ahead "10" blocks
    And the current epoch is "1"
    And the insurance pool balance should be "2400" for the market "ETH/MAR22"

    Then the following transfers should happen:
      | from | to     | from account      | to account             | market id | amount | asset |
      | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_INSURANCE | ETH/MAR22 | 2400   | USD   |
      | lp2  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_INSURANCE | ETH/MAR22 | 0      | USD   |

#auction: (1.0-market.liquidity.priceRange) x min(last trade price, indicative uncrossing price) <=  price levels <= (1.0+market.liquidity.priceRange) x max(last trade price, indicative uncrossing price).



