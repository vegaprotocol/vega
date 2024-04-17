Feature: Test change of SLA market parameter

  Background:
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the following network parameters are set:
      | name                                    | value |
      | market.value.windowLength               | 60s   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 6     |
      | market.auction.minimumDuration          | 1     |
      | market.fee.factors.infrastructureFee    | 0.001 |
      | market.fee.factors.makerFee             | 0.004 |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 20s         | 1              |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 3                 |

    And the liquidity sla params named "SLA-22-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.9         | 0.6                          | 1                             | 1.0                    |
    And the liquidity sla params named "SLA-22-2":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.1         | 0.6                          | 1                             | 1.0                    |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |
    
    And the spot markets:
      | id      | name    | base asset | quote asset | liquidity monitoring | risk model            | auction duration | fees          | price monitoring | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lqm-params           | log-normal-risk-model | 2                | fees-config-1 | price-monitoring | SLA-22     |


    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.7   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |

    Given the average block duration is "1"
  @Now @NoPerp
  Scenario: 001: lp1 and lp2 on the market BTC/ETH, 0044-LIME-091, 0044-LIME-113, 0044-LIME-029, 0044-LIME-115
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | ETH   | 200000 |
      | lp2    | ETH   | 200000 |
      | party1 | ETH   | 100000 |
      | party2 | ETH   | 100000 |
      | party3 | ETH   | 100000 |
      | ptbuy  | ETH   | 100000 |
      | ptsell | ETH   | 100000 |
      | lp1    | BTC   | 200000 |
      | lp2    | BTC   | 200000 |
      | party1 | BTC   | 100000 |
      | party2 | BTC   | 100000 |
      | party3 | BTC   | 100000 |
      | ptbuy  | BTC   | 100000 |
      | ptsell | BTC   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | BTC/ETH   | 4000              | 0.02  | submission |
      | lp_2 | lp2   | BTC/ETH   | 4000              | 0.015 | submission |

    When the network moves ahead "11" blocks

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | BTC/ETH   | 1         | 1                    | buy  | BID              | 12     | 200    | lp-b-1    |
      | lp1   | BTC/ETH   | 1         | 1                    | sell | ASK              | 12     | 200    | lp-s-1    |
      | lp2   | BTC/ETH   | 1         | 1                    | buy  | BID              | 12     | 200    | lp-b-1    |
      | lp2   | BTC/ETH   | 1         | 1                    | sell | ASK              | 12     | 200    | lp-s-1    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 10     | 910   | 0                | TYPE_LIMIT | TIF_GTC | best-buy  |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | BTC/ETH   | sell | 10     | 1110  | 0                | TYPE_LIMIT | TIF_GTC | best-sell |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the opening auction period ends for market "BTC/ETH"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | target stake | supplied stake |
      | 1000       | TRADING_MODE_CONTINUOUS | 8000         | 8000           |
    And the liquidity fee factor should be "0.02" for the market "BTC/ETH"

    ##0044-LIME-091: price range in SLA parameter is getting wider, changes from 0.5 to 0.9
    Then the spot markets are updated:
      | id      | risk model            | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | BTC/ETH | log-normal-risk-model | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22-1   |
    Then the network moves ahead "1" epochs
    And the network treasury balance should be "0" for the asset "ETH"

    Then the network moves ahead "1" epochs
    And the network treasury balance should be "0" for the asset "ETH"
    #0044-LIME-093:price range in SLA parameter is getting narrower, changes from 0.5 to 0.1
    Then the spot markets are updated:
      | id      | risk model            | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | BTC/ETH | log-normal-risk-model | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22-2   |
    Then the network moves ahead "3" epochs

    Then the following transfers should happen:
      | from | to     | from account      | to account                    | market id | amount | asset |
      | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 2400   | ETH   |
      | lp2  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 2400   | ETH   |
    And the network treasury balance should be "6720" for the asset "ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | ptbuy  | BTC/ETH   | buy  | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptsell | BTC/ETH   | sell | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptbuy  | BTC/ETH   | sell | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptsell | BTC/ETH   | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 8000         | 1280           | 3           |

    When the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type   |
      | lp_1 | lp1   | BTC/ETH   | 4000              | 0.02  | amendment |
      | lp_2 | lp2   | BTC/ETH   | 4000              | 0.015 | amendment |

    #0044-LIME-115:during auction the parties place orders within the price range: 0.1 which should count as SLA
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1   | BTC/ETH   | buy  | 12     | 998   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1   | BTC/ETH   | sell | 12     | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp2   | BTC/ETH   | buy  | 12     | 998   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp2   | BTC/ETH   | sell | 12     | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "4" blocks

    #indicative price buy is (990*10+998*24)/34=995; (1010*10+1002*24)/34=1004,
    #last trade price is 1000, so the price range should be: (0.9*995, 1.1*1004)=(895, 1104)
    # (1.0-market.liquidity.priceRange) x min(last trade price, indicative uncrossing price) <=  price levels <= (1.0+market.liquidity.priceRange) x max(last trade price, indicative uncrossing price).
    Then the parties should have the following account balances:
      | party | asset | market id | general | bond |
      | lp1   | ETH   | BTC/ETH   | 171056  | 4000 |
      | lp2   | ETH   | BTC/ETH   | 171088  | 4000 |
    When the network moves ahead "11" blocks
    And the network treasury balance should be "6780" for the asset "ETH"

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | target stake | supplied stake |
      | 994        | TRADING_MODE_CONTINUOUS | 8000         | 8000           |
    Then the parties should have the following account balances:
      | party | asset | market id | general | bond |
      | lp1   | ETH   | BTC/ETH   | 171056  | 4000 |
      | lp2   | ETH   | BTC/ETH   | 171088  | 4000 |


