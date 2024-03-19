Feature: Spot market matching engine

    @MATC
    Scenario: 0068-MATC-060: Any persistent order that is currently ACTIVE or PARKED can be cancelled.

        Given time is updated to "2023-07-20T00:00:00Z"

        Given the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0         | 0                  |
        Given the log normal risk model named "lognormal-risk-model-1":
            | risk aversion | tau  | mu | r   | sigma |
            | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

        And the price monitoring named "price-monitoring-1":
            | horizon | probability | auction extension |
            | 36000   | 0.999       | 3                 |

        And the liquidity sla params named "SLA-1":
            | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
            | 1           | 0.6                          | 2                             | 0.2                    |

        Given the following assets are registered:
            | id  | decimal places |
            | ETH | 1              |
            | BTC | 1              |

        And the following network parameters are set:
            | name                                                | value |
            | network.markPriceUpdateMaximumFrequency             | 2s    |
            | market.liquidity.earlyExitPenalty                   | 0.25  |
            | market.liquidity.bondPenaltyParameter               | 0.2   |
            | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.7   |
            | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
            | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.4   |
            | validators.epoch.length                             | 2s    |
            | limits.markets.maxPeggedOrders                      | 10    |

        And the spot markets:
            | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
            | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | SLA-1      |
        And the following network parameters are set:
            | name                                             | value |
            | market.liquidity.providersFeeCalculationTimeStep | 1s    |
            | market.liquidity.stakeToCcyVolume                | 1     |

        Given the parties deposit on asset's general account the following amount:
            | party  | asset | amount |
            | party1 | ETH   | 10000  |
            | party2 | BTC   | 500    |
            | lp1    | ETH   | 4000   |
            | lp1    | BTC   | 60     |
            | lp2    | ETH   | 4000   |
            | lp2    | BTC   | 60     |

        And the average block duration is "1"

        Given the liquidity monitoring parameters:
            | name               | triggering ratio | time window | scaling factor |
            | updated-lqm-params | 0.2              | 20s         | 0.8            |

        When the spot markets are updated:
            | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
            | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

        When the parties submit the following liquidity provision:
            | id  | party | market id | commitment amount | fee | lp type    |
            | lp1 | lp1   | BTC/ETH   | 3000              | 0.1 | submission |

        Then the network moves ahead "1" blocks

        Then the market data for the market "BTC/ETH" should be:
            | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
            | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 2400         | 3000           | 0             |

        #0068-MATC-084: GFN order rejected
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | expires in | error |
            | party1 | BTC/ETH   | buy  | 4      | 8     | 0                | TYPE_LIMIT | TIF_GTC | party-order5 |            |       |
            | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |            |       |
            | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |            |       |
            | party2 | BTC/ETH   | sell | 6      | 24    | 0                | TYPE_LIMIT | TIF_GTC | party-order6 |            |       |

        When the network moves ahead "2" blocks

        Then the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 15    | 1    | party2 |

        Then the market data for the market "BTC/ETH" should be:
            | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
            | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 14        | 17        |

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  | expires in | error |
            | party1 | BTC/ETH   | buy  | 1      | 8     | 0                | TYPE_LIMIT | TIF_GFN | party1-GFN |            |       |
            | party1 | BTC/ETH   | buy  | 1      | 8     | 0                | TYPE_LIMIT | TIF_GTT | party1-GTT | 10         |       |

        #trigger auction
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | BTC/ETH   | buy  | 1      | 13    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | BTC/ETH   | sell | 1      | 13    | 0                | TYPE_LIMIT | TIF_GTC |

        Then the market data for the market "BTC/ETH" should be:
            | mark price | trading mode                    | auction trigger       |
            | 15         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

        #0068-MATC-084: GFN order rejected
        #0068-MATC-085: GTT order stays ACTIVE
        Then the orders should have the following status:
            | party  | reference  | status           |
            | party1 | party1-GFN | STATUS_CANCELLED |
            | party1 | party1-GTT | STATUS_ACTIVE    |

        #0068-MATC-080: orders are placed into the book and no matching takes place.
        #0068-MATC-081: post only orders are placed into the book and no matching takes place.
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  | only |
            | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GFA | party1-GFA |      |
            | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC |            |      |
            | party1 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC |            |      |
            | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC |            | post |
            | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC |            | post |

        And the parties place the following pegged orders:
            | party | market id | side | volume | pegged reference | offset | reference |
            | lp1   | BTC/ETH   | buy  | 2      | BID              | 3      | lp1-peg-b |
            | lp1   | BTC/ETH   | sell | 2      | ASK              | 3      | lp1-peg-s |

        #0068-MATC-083: pegged order is parked during auction
        Then the orders should have the following status:
            | party  | reference  | status        |
            | lp1    | lp1-peg-b  | STATUS_PARKED |
            | lp1    | lp1-peg-s  | STATUS_PARKED |
            | party1 | party1-GFA | STATUS_ACTIVE |

        When the parties cancel the following orders:
            | party  | reference  |
            | lp1    | lp1-peg-b  |
            | party1 | party1-GFA |
        Then the orders should have the following status:
            | party  | reference  | status           |
            | lp1    | lp1-peg-b  | STATUS_CANCELLED |
            | lp1    | lp1-peg-s  | STATUS_PARKED    |
            | party1 | party1-GFA | STATUS_CANCELLED |
